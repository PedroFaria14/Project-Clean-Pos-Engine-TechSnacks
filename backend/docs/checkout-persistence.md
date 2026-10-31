# Estados persistidos e idempotência do checkout

## Contrato e comportamento

O contrato HTTP está em `api/openapi.yaml`.

Após a implantação de autenticação, todas as rotas abaixo exigem sessão válida; mutações também exigem Origin permitido e CSRF. As chaves novas são separadas por usuário e pedidos legados sem responsável são restritos a administradores. Consulte [autenticação e operação](authentication.md).

- `POST /checkout` aceita o header opcional `Idempotency-Key` (1–128 caracteres ASCII: letras, números, `_` e `-`).
- A mesma chave e os mesmos campos reconhecidos do pedido devolvem a resposta original, inclusive preço e status originais. A ordem das propriedades JSON e os espaços não importam; a ordem dos itens importa.
- Outro conteúdo com a mesma chave retorna `409`. Um `order_id` já existente também retorna `409` quando não corresponde a uma repetição da mesma tentativa.
- O caminho com chave usa uma transação para pedido, itens e registro idempotente. A chave é única no banco e a tentativa perdedora sofre rollback antes de recuperar o resultado vencedor.
- Erros de consulta/gravação da idempotência retornam `503` sem detalhes do banco. Reenvie a mesma tentativa, não gere outra chave.
- Sem chave, não há garantia de deduplicação. Falhas de dependência agora retornam `503`, sem detalhes internos; erros de domínio permanecem em `422`.
- Não há expiração automática de chaves. Remover registros idempotentes remove a garantia de deduplicação correspondente.
- O registro é específico de `POST /checkout`. Uma chave não substitui autenticação ou autorização: a API usa a sessão para determinar o escopo do usuário.
- `GET /orders/{id}` apresenta o status atual persistido (`OPEN`, `AWAITING_PAYMENT` ou `COMPLETED`). A recuperação não executa checkout/pagamento nem notifica observadores.
- O valor total persistido é um snapshot, assim como os nomes e preços dos itens; a consulta não depende do catálogo atual.

## Migration explícita

A API verifica a versão do schema ao iniciar. O repositório de pedidos não executa mais `AutoMigrate`.
O catálogo também usa migration explícita a partir de `000003_product_catalog.up.sql`; seu repositório não executa mais `AutoMigrate`.

Antes de atualizar um banco existente, faça backup e confira o destino definido por `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` e `DB_NAME`.
O comando abaixo aplica as migrations **ao banco configurado nessas variáveis**; ele não é executado automaticamente pela API:

```powershell
# A partir de backend/, com as variáveis DB_* configuradas:
go run ./cmd/migrate
go run ./cmd/api
```

Para o ambiente Compose já configurado, depois de disponibilizar o banco:

```powershell
docker compose build backend
docker compose run --rm --no-deps backend /usr/local/bin/migrate
docker compose up -d backend frontend
```

A migration inicial:

1. Cria as tabelas de pedidos/itens somente quando ainda não existem.
2. Acrescenta o status e preenche pedidos legados sem status como `AWAITING_PAYMENT`, conforme regra aprovada. Não presume pagamento confirmado.
3. Cria a tabela de resultados idempotentes com chave única e referência ao pedido.
4. Registra a versão em `techsnacks_schema_migrations`. Reexecuções não reaplicam versões concluídas.

O runner usa uma transação e advisory lock para serializar migrations concorrentes.

O arquivo `.down.sql` é um rollback **manual** da primeira migration, a executar em transação e sem instâncias da API gravando no banco. Ele bloqueia execução quando existem migrations dependentes (incluindo autenticação). Quando aplicável, preserva pedidos e itens, mas remove status e histórico de deduplicação; não recupera esses metadados após sua remoção. Não execute rollback fora de ordem.

## Recuperação no frontend

A tentativa (chave, usuário e itens) é gravada no `sessionStorage`, separada por conta, antes do envio. Após falha, reload ou expiração da sessão, a mesma conta pode selecionar **Retomar pedido pendente**, usando exatamente a mesma chave e o mesmo conteúdo. Os itens ficam bloqueados até resolver ou descartar explicitamente a tentativa. Registros v1 anônimos exigem conferência administrativa e não são reenviados automaticamente.

**Iniciar nova venda** descarta somente o registro local, mediante confirmação; não cancela um pedido no servidor. Armazenamento corrompido/inacessível não é descartado silenciosamente. Se a gravação local falhar, a primeira requisição não é enviada.

O armazenamento é por sessão/aba do navegador, não um backup permanente: fechar a sessão, limpar dados ou trocar de navegador pode perder a chave. Não há retries automáticos nem integração de pagamento neste fluxo.

## Testes

```powershell
# backend/
go test ./...
go vet ./...

# frontend/ (Node.js 22.18+ ou 24 para executar testes TypeScript nativos)
npm run test:checkout
npm run build
```

### PostgreSQL descartável

Os testes de integração exigem PostgreSQL real. Não usam SQLite como substituto das garantias de concorrência do PostgreSQL.
O exemplo abaixo cria um container sem volumes, separado do Compose da aplicação:

```powershell
docker run --detach --rm --name techsnacks-checkout-test -e POSTGRES_PASSWORD=test-only -e POSTGRES_DB=techsnacks_test_checkout -p 127.0.0.1:55432:5432 postgres:15-alpine
docker exec techsnacks-checkout-test pg_isready -U postgres -d techsnacks_test_checkout

# Depois que pg_isready confirmar a disponibilidade, em backend/:
$env:TECHSNACKS_TEST_DATABASE_URL = 'postgres://postgres:test-only@127.0.0.1:55432/techsnacks_test_checkout?sslmode=disable'
go test -tags=integration -run TestPostgres -v ./internal/infra/repository

# Encerrar somente o container de testes criado acima:
docker stop techsnacks-checkout-test
```

A suíte aceita somente host de loopback e banco com prefixo `techsnacks_test_`. Cada teste cria e remove seu próprio schema aleatório. Os cenários verificam migration legada/rollback, restauração de estados, gravações concorrentes, conflito de conteúdo e rollback de pedido/itens quando a gravação idempotente falha.

Sem `TECHSNACKS_TEST_DATABASE_URL`, os testes são sinalizados como `SKIP`; isso não equivale a uma homologação da integração.
