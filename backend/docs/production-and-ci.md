# Infraestrutura, CI e homologação real

## Ambientes

| Arquivo | Finalidade |
| --- | --- |
| `docker-compose.yml` | Desenvolvimento; conserva `pgdata` e usa Vite |
| `compose.test.yml` | Stack descartável, sem volumes da aplicação |
| `compose.production.yml` | Runtimes não-root, banco externo com TLS e frontend em loopback |
| `compose.production.tools.yml` | Migrations e criação de contas com credencial administrativa separada |
| `deploy/Caddyfile` | HTTPS no host, à frente do frontend; exige domínio, DNS e operação do Caddy |

Nada faz deploy, push de imagem, migration em banco existente ou cria contas reais automaticamente.

## Desenvolvimento

Na raiz, com Docker disponível:

```powershell
docker compose build
docker compose up -d --wait db
docker compose run --rm --no-deps backend /usr/local/bin/migrate
# Crie a conta administrativa conforme authentication.md, usando /usr/local/bin/create_user no container.
docker compose up -d --wait backend frontend
```

Frontend: `http://localhost:5173`. As portas de desenvolvimento estão em loopback. `docker compose down` preserva os dados; não use `down --volumes` na aplicação.

O backend contém binários `api`, `migrate` e `create_user`, não a toolchain Go: comandos antigos `backend go run ...` devem usar esses binários. O frontend Docker tem targets `development` (Vite) e `runtime` (build estático/Nginx).

A migration `000003_product_catalog.up.sql` preserva o catálogo existente e cria sua tabela em bancos novos. Não existe mais `AutoMigrate` nos repositórios. A API exige explicitamente o schema atualizado antes de iniciar.

## Servidor e PostgreSQL

Timeouts HTTP: cabeçalhos 5 s, leitura 15 s, escrita 30 s, idle 60 s, cabeçalhos até 1 MiB e encerramento gracioso de 15 s. Containers têm 20 s para terminar. `HTTP_ADDR` altera `:8080`; probes das imagens pressupõem a porta interna padrão.

| Variável | Contrato |
| --- | --- |
| `DB_HOST`, `DB_USER`, `DB_NAME` | Obrigatórias |
| `DB_PORT` | `5432` |
| `DB_PASSWORD` / `DB_PASSWORD_FILE` | Alternativas mutuamente exclusivas |
| `DB_SSLMODE` | `disable` no desenvolvimento; somente `verify-full` em produção |
| `DB_SSLROOTCERT` | CA do PostgreSQL, quando aplicável |
| `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS` | `20` / `5`; idle não pode exceder open |
| `DB_STATEMENT_TIMEOUT` | `15s`; ferramentas administrativas de produção usam `5m` |
| `DB_SEARCH_PATH` | Schema opcional validado, usado para isolamento de testes |

Credenciais são codificadas na URI e não aparecem nos erros. O pool fecha no encerramento, recicla conexões em 30 minutos e idle em 5 minutos. Logs do GORM não imprimem SQL, argumentos ou respostas brutas de dependências.

`GET /health/live` mede apenas o processo; `GET /health/ready` verifica PostgreSQL com deadline de 2 s. Ambos são públicos, sem dados de negócio; falhas de readiness retornam `503` sem diagnóstico interno. Readiness não equivale a auditoria dos dados ou teste de capacidade.

Cadastro valida campos com `400` + `details`, retorna `409` para ID repetido e não sobrescreve o produto. Falhas de infraestrutura em catálogo/checkout retornam `503`; erros de domínio do checkout continuam em `422`.

## Testes reproduzíveis

Pré-requisitos: Go 1.26.10+, Node 24, npm e PostgreSQL 15 descartável para a suíte real.

```powershell
# frontend/
npm ci
npx --no-install playwright install chromium
npm test
npm run lint
npm run build
npm run test:e2e:mock
```

Os 34 cenários de navegador estão versionados em `frontend/e2e/mock/`; não dependem mais de arquivos temporários do agente. Usam mocks para falhas, estados de UI e acessibilidade, não para provar conectividade real.

Disponibilize um banco local descartável, com nome `techsnacks_test_*`. Exemplo sem volume:

```powershell
docker run --detach --rm --name techsnacks-e2e-db -e POSTGRES_PASSWORD=test-only -e POSTGRES_DB=techsnacks_test_local -p 127.0.0.1:55432:5432 postgres:15-alpine
docker exec techsnacks-e2e-db pg_isready -h 127.0.0.1 -U postgres -d techsnacks_test_local
# Depois de confirmar disponibilidade, em frontend/:
$env:TECHSNACKS_E2E_DATABASE_URL = 'postgres://postgres:test-only@127.0.0.1:55432/techsnacks_test_local?sslmode=disable'
npm run test:e2e
# Encerrar somente o container de testes criado acima:
docker stop techsnacks-e2e-db
```

O runner cria um schema próprio aleatório, compila os três comandos Go, aplica migrations nesse schema, cria contas sintéticas com senha aleatória, inicia a API e serve o build estático por um proxy same-origin. Executa Playwright **sem mocks de API**, encerra processos e remove somente o schema criado. Recusa URLs externas e bancos sem prefixo de teste. O usuário de teste precisa poder criar/remover schemas.

As jornadas reais verificam cookie HttpOnly, login/logout/revogação, CRUD, reload de rota SPA, CSRF, papéis, acesso a pedido de outro usuário, retomada idempotente e valores históricos. O proxy Node é somente de teste, não substitui o Nginx de produção.

Para Chrome já instalado no Windows, use `PLAYWRIGHT_CHANNEL=chrome`. O CI instala Chromium. `npm run test:e2e:external` exige `E2E_DATABASE_URL`, `E2E_SCHEMA`, `E2E_BASE_URL` local e `E2E_PASSWORD`; destina-se à stack `compose.test.yml`. Seus testes truncam tabelas do schema de teste explícito, nunca devem apontar para produção.

## CI e artifacts

`.github/workflows/ci.yml` roda em push, PR e execução manual, com permissões de leitura e sem secrets de produção:

1. Go: formatação, vet, testes unitários/integração PostgreSQL com `-race`, build e `govulncheck`.
2. Frontend: `npm ci`, unitários, lint, build, audit e os 34 cenários mockados.
3. Imagens: build multi-stage, migrations/contas no banco descartável, UID não-root, probes e jornadas reais através do **Nginx das imagens**, não do proxy Node.
4. Trivy bloqueia HIGH/CRITICAL. Gera SBOMs CycloneDX e disponibiliza imagens aprovadas, etiquetadas pelo SHA do commit, como artifacts por 7 dias. Não publica em registry ou faz deploy.
5. Backup/restore usa `pg_dump`/`pg_restore` da própria imagem PostgreSQL 15, outro banco descartável e uma linha de verificação.

Configure os jobs como checks obrigatórios na proteção de branch. Dependabot propõe atualizações semanais, sem merge automático. Actions são fixadas em SHA. Um workflow novo precisa executar no GitHub: validação estática não equivale a um pipeline verde.

## Produção e privilégio mínimo

Copie `deploy/.env.example` para `deploy/.env.production` (ignorado pelo Git). Use arquivos de senha fora do repositório, legíveis pelo UID 10001 dos comandos Go e restritos no host.

Use imagens aprovadas do CI (linux/amd64), carregadas com `docker load --input runtime-images.tar` ou publicadas por um processo de release controlado. Configure tags SHA/digests, não `latest`. Para outra arquitetura, construa e valide imagens correspondentes.

Provisione PostgreSQL externo com TLS e certificado válido. Separe o migrator/administrador do usuário runtime, que não deve ser proprietário nem poder criar objetos ou contas. O runtime precisa de:

- SELECT em `users` e `techsnacks_schema_migrations`.
- SELECT/INSERT/UPDATE/DELETE em `products`, `order_models`, `order_item_models`, `checkout_idempotency_records`, `user_sessions`, `auth_login_limits`.
- USAGE no schema e USAGE/SELECT na sequência dos itens.

O administrador aplica os grants após a migration, conforme o schema do ambiente. A senha administrativa não é montada nos serviços runtime.

```powershell
# Confira destino e backup antes de usar credenciais administrativas:
docker compose --env-file deploy/.env.production -f compose.production.tools.yml run --rm migrate
# Com TECHSNACKS_USER_PASSWORD fornecida no terminal pela entrada oculta de authentication.md:
docker compose --env-file deploy/.env.production -f compose.production.tools.yml run --rm -e TECHSNACKS_USER_PASSWORD create-user /usr/local/bin/create_user --username admin.local --role admin
docker compose --env-file deploy/.env.production -f compose.production.yml up -d --wait
```

Instale/opere Caddy no host com `APP_DOMAIN` e `ACME_EMAIL`, valide `deploy/Caddyfile` com `caddy validate` e configure DNS/portas 80/443. Caddy gerencia HTTPS, consulta readiness e encaminha para `127.0.0.1:8081`. O backend não tem porta publicada.

Go usa UID 10001 e Nginx UID 101; root filesystem readonly, capabilities removidas e sem ganho de privilégio. Nginx usa tmpfs limitado, CSP/nosniff/anti-frame, cache de assets e fallback SPA; DNS Docker acompanha recriações do backend. Memória padrão: Go 512 MiB e Nginx 128 MiB, ajustáveis após medição.

## Operação, backup e rollback

- Monitore readiness, taxa de 5xx, latência, CPU/memória e saturação do pool. A stack não instala Prometheus/Grafana/OTel ou alertas de provedor automaticamente.
- Readiness `503` com liveness `200`: investigue banco, TLS e pool; não reinicie em loop por toda indisponibilidade de dependência. Diagnósticos detalhados devem vir de logs autorizados do PostgreSQL, não das respostas públicas.
- Mantenha backups criptografados fora do host, com retenção e acesso definidos. O drill no CI usa dados sintéticos: não prova restauração de um backup real do estabelecimento.
- Antes de atualizar, faça backup verificado e registre SHA/digest anterior. Leia migrations e não reverta schema automaticamente. Em incidente, retire tráfego, avalie compatibilidade, volte às imagens aprovadas e confirme readiness e uma jornada em homologação.
- Rotacionar senha do banco exige atualizar secret e recriar instâncias controladamente. Sessões persistem no banco; sair revoga a sessão atual.

## Scans e limites

Versões sinalizadas de Go, pgx, x/net, x/text e npm foram atualizadas. Mantenha go.mod, builder e CI sincronizados nas próximas atualizações de Go.

`govulncheck` ainda pode listar o aviso sem correção de `golang.org/x/crypto/openpgp`: esse pacote não é importado/usado; a aplicação usa Argon2id, e o scan não encontra símbolos ou pacotes importados atingidos. Não existe ignore ocultando o aviso. Reavalie se uma integração passar a usar OpenPGP.

Docker indisponível no host impede executar builds/smoke/scans das imagens e o drill de restore localmente. Configs validadas e testes nativos não substituem isso. O job `images-and-e2e` deve passar antes de uso em produção. TLS/DNS/CA reais também dependem do ambiente do operador.
