# Autenticação e permissões

## Contrato

Consulte `api/openapi.yaml`. As rotas de negócio agora exigem uma sessão válida.

| Endpoint | Comportamento |
| --- | --- |
| `POST /sessions` | Login; retorna `201`, atributos públicos, expiração e token CSRF; define cookie de sessão |
| `GET /sessions/current` | Consulta a sessão válida e recupera o token CSRF |
| `DELETE /sessions/current` | Revoga a sessão atual e remove o cookie (`204`) |

Não há cadastro público ou usuário/senha padrão. Uma falha de login não informa se a conta existe.

| Operação | Operador | Administrador |
| --- | --- | --- |
| Consultar catálogo e registrar venda | Sim | Sim |
| Consultar pedido próprio | Sim | Sim |
| Consultar pedido de outra conta ou pedido legado sem responsável | Não (`404`) | Sim |
| Cadastrar/excluir produtos | Não (`403`) | Sim |

O backend verifica permissões; a restrição de menu no frontend é apenas apresentação. Papel e usuário enviados pelo cliente não concedem privilégios.

## Configuração

As variáveis `DB_*` continuam definindo o banco. A aplicação não carrega arquivos `.env` automaticamente.

O pool, TLS obrigatório do PostgreSQL em produção, binários dos containers e ferramentas administrativas com credencial separada estão detalhados em [produção e CI](production-and-ci.md).

| Variável | Uso |
| --- | --- |
| `APP_ENV` | `development` (padrão local) ou `production` |
| `APP_ALLOWED_ORIGINS` | Origens completas, sem barra final, separadas por vírgula; obrigatória em produção |
| `AUTH_TRUSTED_PROXIES` | IPs/CIDRs dos proxies realmente controlados, separados por vírgula; nenhum é confiável por padrão |
| `API_PROXY_TARGET` | Variável **do servidor Vite**, não do bundle; padrão local `http://127.0.0.1:8080`, Compose `http://backend:8080` |

Em desenvolvimento, as origens padrão são `http://localhost:5173` e `http://127.0.0.1:5173`. O Compose é um ambiente de desenvolvimento, não um manifesto de produção.

Em produção, configure `APP_ENV=production`, origens HTTPS explícitas e publique frontend e `/api` sob a mesma origem HTTPS com reverse proxy. A API recusa origens HTTP na configuração de produção, usa cookies `Secure` e desliga o modo debug do Gin. O servidor Vite é somente para desenvolvimento. Habilitar `Secure` não substitui a configuração de TLS no proxy.

O cookie `techsnacks_session` é host-only, `HttpOnly`, `SameSite=Lax`, `Path=/`, válido por 8 horas e não é renovado implicitamente. O identificador tem 256 bits aleatórios; somente seu SHA-256 é persistido no banco. Logout revoga apenas a sessão atual. Contas desativadas perdem acesso nas próximas requisições, sem cache de autorização em memória.

As senhas são codificadas em Argon2id PHC, com salt aleatório de 16 bytes, memória de 64 MiB, 3 iterações, paralelismo 1 e chave de 32 bytes. A verificação limita parâmetros para rejeitar hashes corrompidos com custos ilimitados. O serviço permite no máximo duas verificações KDF simultâneas por instância; dimensione memória/latência no hardware de destino. O formato permite evolução futura de parâmetros; não há fluxo de alteração/recuperação de senha nesta entrega.

## Aplicar migration e criar contas

A migration `000002_authentication.up.sql` adiciona usuários, sessões, limites de login, responsável pelo pedido e espaço para as chaves internas por usuário. Não atribui pedidos antigos a uma conta arbitrária.

Depois de conferir o banco de destino e fazer backup, em `backend/`:

```powershell
go run ./cmd/migrate
```

Para criar uma conta, forneça uma senha de **12 a 128 bytes UTF-8** somente ao comando administrativo. A senha não é aceita por argumento de linha de comando, impressa ou salva em arquivo pelo comando. Exemplo PowerShell com entrada oculta:

```powershell
$secure = Read-Host 'Senha da nova conta' -AsSecureString
$pointer = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)
try {
    $env:TECHSNACKS_USER_PASSWORD = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($pointer)
    go run ./cmd/create_user --username admin.local --role admin
} finally {
    [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($pointer)
    Remove-Item Env:TECHSNACKS_USER_PASSWORD -ErrorAction SilentlyContinue
}
```

Use o mesmo comando com `--role operator` e outro nome para operadores. Nomes são normalizados para minúsculas e aceitam 3 a 64 letras ASCII, dígitos, ponto, `_` e `-`. A criação falha se a conta já existir; não sobrescreve senhas ou permissões silenciosamente. Proteja o acesso ao host e ao banco: esse comando administrativo não é uma API pública.

Não reverta a aplicação para uma versão anônima como rollback de autenticação: isso voltaria a expor dados. A migration de pedidos anterior bloqueia rollback fora de ordem quando há migrations dependentes. Uma reversão exige planejamento de versão, backup e controle de acesso; não há rollback automático removendo usuários.

## CSRF, CORS e limitação de login

- `POST /sessions` exige `Origin` pertencente à allowlist, inclusive para clientes de teste/CLI.
- Mutações autenticadas exigem `Origin` permitido e `X-CSRF-Token`, obtido ao criar/consultar a sessão. O token é derivado da sessão por HMAC e permanece apenas em memória no frontend.
- O frontend usa `/api`, cookies same-origin e cliente HTTP centralizado. Não grava cookies, senhas ou tokens de autenticação em `localStorage`/`sessionStorage`.
- Login permite 10 tentativas por conta e 100 por endereço em janelas de 15 minutos. Todas as tentativas contam; a resposta `429` inclui `Retry-After: 900`.
- Os contadores e suas atualizações atômicas ficam no PostgreSQL e funcionam entre instâncias. Endereços/contas são registrados nos contadores como hashes, não em claro; hashes de IP continuam sendo dados pseudonimizados, não anonimização garantida. Contadores com mais de um dia são removidos no próximo login.
- Por padrão, o endereço é o da conexão. Atrás de proxy não confiável/configurado, vários operadores podem compartilhar esse limite. Configure `AUTH_TRUSTED_PROXIES` somente com proxies controlados; cabeçalhos encaminhados de clientes arbitrários não devem definir o endereço para rate limiting.
- As respostas não são cacheadas. Logs HTTP registram método, rota-modelo, status, duração e request ID, sem senha, cookie, token, nome de usuário, parâmetros de URL ou IP. Consultas do repositório de autenticação não imprimem argumentos SQL.

## Recuperação do checkout entre contas

Novos pedidos recebem `created_by`. A chave interna de deduplicação usa `user:<ID>:<Idempotency-Key>`; o header público continua com limite de 128 caracteres. A mesma conta pode retomar a venda após novo login, sem compartilhar o resultado com outra conta.

O frontend guarda apenas a tentativa de venda (não credenciais) em `techsnacks.checkout.pending.v2:<ID>`. Sair não descarta a tentativa. Uma conta diferente tem seu próprio registro local.

Tentativas v1 anteriores à autenticação não possuem responsável verificável. Elas não são associadas nem reenviadas automaticamente por uma conta nova. O frontend bloqueia nova venda até um administrador conferir os registros no servidor e descartar explicitamente o registro antigo. O descarte local não cancela pedidos no servidor.

## Verificação

```powershell
# backend/
go test ./...
go vet -tags=integration ./...

# Com PostgreSQL local descartável e TECHSNACKS_TEST_DATABASE_URL configurada:
go test -tags=integration -run TestPostgres -v ./internal/infra/repository

# frontend/
npm run test:checkout
npm run lint
npm run build
```

Os testes de PostgreSQL usam schemas isolados e verificam login, segredo persistido apenas como hash, sessão expirada/revogada, CSRF, Origin, privilégios, isolamento de pedidos/chaves e limite de login concorrente. Consulte `checkout-persistence.md` para provisionar um PostgreSQL descartável.

Esses controles não equivalem a uma auditoria adversarial. Antes de exposição pública, valide TLS/reverse proxy, infraestrutura, backups, gestão de contas e dependências no ambiente de destino; uma revisão especializada de segurança é recomendada.
