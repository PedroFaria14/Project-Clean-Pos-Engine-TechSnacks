# Neon: URL completa e ambiente local/Render

O backend aceita a conexão completa em **`DATABASE_URL`** ou no alias **`Data_Base`** (com essa grafia exata no Linux/Render). Ambos usam o mesmo parser. Prefira apenas uma variável; se ambas existirem com valores diferentes, a inicialização falha sem imprimir as URLs.

Exemplo fictício em `backend/.env`:

```dotenv
APP_ENV=development
Data_Base="postgresql://USER:PASSWORD@HOST-DO-NEON/neondb?sslmode=require&channel_binding=require"
DB_SSLMODE=verify-full
```

Não publique a URL real. `.env` é ignorado pelo Git e excluído das imagens. `backend/.env.example` contém somente placeholders.

## Carregamento local

Os comandos `api`, `migrate`, `create_user`, `dbcheck` e `healthcheck` carregam `.env` local quando ainda não há conexão configurada no ambiente. A execução deve ocorrer em `backend/`, ou a partir da raiz com os binários apropriados (também há fallback para `backend/.env`).

O carregamento não sobrescreve variáveis existentes. Em produção, no Render ou quando `DATABASE_URL`, `Data_Base` ou `DB_HOST` já estão definidos, o arquivo não é usado como fallback. A carga ocorre antes da configuração de autenticação, para não aplicar cookies de desenvolvimento a uma configuração local explicitamente marcada como produção.

Use **uma** forma de identificar a conexão:

- URL em `DATABASE_URL` ou `Data_Base`; ou
- campos `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_NAME`, `DB_PASSWORD`/`DB_PASSWORD_FILE`.

Não misture a URL com esses campos separados. `DB_SSLMODE`, `DB_SSLROOTCERT`, limites de pool e timeout continuam sendo configurações opcionais compatíveis com a URL.

## TLS e parâmetros Neon

Usuário, senha e database são decodificados corretamente. Parâmetros da URL, incluindo `channel_binding=require`, são preservados. Identidade duplicada em query params e parâmetros repetidos são recusados para evitar configurações ambíguas.

Em produção, uma URL do provedor com `sslmode=require` é fortalecida para **`verify-full`**. Não há downgrade de verificação de certificado; URLs que desativam TLS e overrides inseguros são recusados em produção. No desenvolvimento, use `DB_SSLMODE=verify-full` para verificar também certificado/hostname.

O endpoint com `-pooler` pode ser usado pela API. Para migrations e tarefas administrativas, prefira o endpoint direto fornecido pelo Neon. Alterar esse destino é uma configuração explícita; a aplicação não reescreve silenciosamente o hostname do usuário.

## Testar apenas a conexão

```powershell
# backend/
go run ./cmd/dbcheck
```

Esse comando carrega a configuração, conecta e executa `SELECT 1` em uma transação read-only, finalizada com rollback. Não verifica/cria schema, não executa migrations e não cria contas. O resultado não imprime host, usuário, senha ou URI. A imagem também inclui `/usr/local/bin/dbcheck`.

Uma conexão válida **não** significa que a API já pode iniciar: o schema precisa estar preparado. Migrations e criação de administrador são operações separadas, explícitas, a executar somente após conferir o banco de destino e os dados existentes.

## No Render

Em **Environment → Add Environment Variable**, configure:

| Key | Value |
| --- | --- |
| `Data_Base` | URL completa copiada do Neon, incluindo senha e parâmetros |
| `APP_ENV` | `production` |
| `PORT` | `10000` (ou a porta fornecida pela plataforma) |
| `APP_ALLOWED_ORIGINS` | URL HTTPS real do frontend, sem barra final |
| `DB_SSLMODE` | `verify-full` |

Você pode usar `DATABASE_URL` no lugar de `Data_Base`. Ao trocar uma configuração antiga por URL única, remova `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_NAME`, `DB_PASSWORD` e `DB_PASSWORD_FILE` do serviço. Render preserva variáveis removidas de um Blueprint; a limpeza deve ser explícita no painel.

O template `render.yaml` solicita `Data_Base`. Não existe conexão automática ao arquivo `.env` da sua máquina no Render, nem banco Neon criado pelo template. Após salvar as variáveis, faça o redeploy.

Consulte `render.md` para Root Directory, Dockerfile, migrations e proxy. `frontend/.env` e variáveis `VITE_*` nunca devem receber essa URL, pois credenciais de banco são exclusivas do backend.
