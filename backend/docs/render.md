# Deploy no Render: API, PostgreSQL e frontend

O projeto usa serviços separados: API Go, PostgreSQL e frontend. Render não executa o Compose ao criar um Web Service Docker. Nenhum recurso é publicado, banco modificado ou conta real criada automaticamente por este guia ou pelo agente.

## 1. Código e validação

Envie ao GitHub um commit com todos os arquivos necessários, incluindo migrations, Dockerfiles, script de runtime e testes. Espere os checks do CI antes de configurar auto-deploy como **After CI Checks Pass**. Os builds das imagens e scans precisam passar; validação estática não equivale a isso.

`render.yaml` contém um template **somente da API**, com plano Free, região Ohio e variáveis solicitadas pelo painel. Não cria o banco. Ajuste nome/região antes de aplicar; prefira o fluxo manual abaixo para o primeiro deploy e para obter as URLs reais. Free é adequado para demonstração e pode suspender a instância por inatividade; não é uma garantia de disponibilidade de produção.

## 2. Banco PostgreSQL

Crie ou selecione um PostgreSQL, preferencialmente na mesma região da API. Os testes do projeto usam PostgreSQL 15; valide outra major antes de escolhê-la para produção.

Copie a URL completa para `DATABASE_URL` ou `Data_Base`, ou configure hostname **externo completo**, porta, database, username e senha separadamente em `DB_*`. Não misture as duas formas. Para Neon e `.env` local, consulte [neon.md](neon.md). Não use `localhost`, `db` ou um endereço IP resolvido anteriormente. Senhas nos campos separados são fornecidas em texto original; na URL, preserve o encoding fornecido pelo provedor.

### TLS e a URL interna do Render

A política da aplicação exige `verify-full` em produção. A documentação do Render informa que conexões internas usam certificados autoassinados e não suportam `verify-full`/`verify-ca` (suportam `require`). Portanto, o template preserva a política atual e usa o **hostname externo**.

Valide o certificado/hostname na conexão externa; a imagem inclui CAs públicas. Se houver falha de certificado, use a CA apropriada em `DB_SSLROOTCERT`, disponibilizada como arquivo secreto. Não configure o caminho do Compose `/run/secrets/db_ca` no Render sem montar esse arquivo. Não desabilite TLS para contornar o erro.

Conexões externas precisam estar autorizadas para seu computador durante a preparação e para o serviço no deploy. Restrinja a lista de IPs conforme as regras do ambiente. Não é necessário publicar a senha no repositório ou enviá-la pelo chat.

## 3. Preparar schema e administrador antes da API

A API verifica a versão do schema antes de abrir a porta. Um banco novo sem migrations fará o deploy falhar com a mensagem de schema não preparado.

O plano Free não oferece shell/one-off jobs e o pre-deploy command é um recurso de serviços pagos. Nesse plano, prepare o banco do seu computador, usando a conexão externa. No terminal, configure `APP_ENV=production` e os `DB_*` do banco Render escolhido, sem `DB_SEARCH_PATH` (schema público padrão):

```powershell
# Em backend/, com as variáveis DB_* configuradas explicitamente:
go run ./cmd/migrate
# Com TECHSNACKS_USER_PASSWORD fornecida pela entrada oculta de authentication.md:
go run ./cmd/create_user --username admin.local --role admin
```

Os comandos acima **modificam o banco configurado**; confirme destino e backup antes de executá-los. Use uma credencial administrativa para preparar schema/contas e uma credencial de privilégios mínimos no runtime, conforme `production-and-ci.md`. Não execute migrations no `RUN` do Dockerfile ou implicitamente a cada despertar da instância Free.

Em um serviço pago, a migration pode ser uma etapa controlada de pré-deploy com `/usr/local/bin/migrate`, se a credencial fornecida tiver os privilégios adequados. O template Free não configura essa opção.

## 4. Web Service da API

| Campo | Valor |
| --- | --- |
| Name | Um nome único, por exemplo `techsnacks-api` |
| Language | Docker |
| Branch | Seu commit validado em `main` |
| Region | Região do banco |
| Root Directory | `backend` |
| Dockerfile Path | `./Dockerfile` (relativo ao Root Directory) |
| Docker Build Context | `.` |
| Docker Command | Vazio, usando o CMD da imagem |
| Health Check Path | `/health/ready` |

Variáveis:

```dotenv
APP_ENV=production
PORT=10000
APP_ALLOWED_ORIGINS=https://URL-REAL-DO-FRONTEND
DB_HOST=HOSTNAME-EXTERNO-COMPLETO-DO-POSTGRES
DB_PORT=5432
DB_NAME=DATABASE-DO-PAINEL
DB_USER=USUARIO-DO-PAINEL
DB_PASSWORD=SEGREDO-CONFIGURADO-NO-PAINEL
DB_SSLMODE=verify-full
```

Substitua os valores pelos dados do seu ambiente; as strings acima são placeholders. `APP_ALLOWED_ORIGINS` é uma lista separada por vírgulas de origens HTTPS exatas, sem barra final. Enquanto o frontend não existir, a própria origem HTTPS da API pode ser configurada para testes de clientes HTTP; atualize para a URL real do frontend antes de testar login nele.

Não configure `DB_PASSWORD_FILE` junto com `DB_PASSWORD`. Não configure `DB_SEARCH_PATH` de testes. `HTTP_ADDR`, quando definido, tem precedência; deixe-o ausente no Render para a aplicação usar automaticamente `0.0.0.0:$PORT`. Não escreva `${PORT}` literalmente em seu valor: não há interpolação de shell no código Go.

O healthcheck da imagem usa a mesma resolução de porta da API, inclusive `PORT=10000`. O runtime também inclui `/usr/local/bin/healthcheck`. A raiz `/` da API pode retornar `404`; verifique `/health/live` e `/health/ready`.

## 5. Web Service do frontend

Use um Web Service **Docker**, para servir a SPA e encaminhar `/api` no mesmo domínio. Um Static Site isolado não executa nosso proxy Nginx, e apenas trocar uma URL no JavaScript não resolve cookies/CSRF.

| Campo | Valor |
| --- | --- |
| Root Directory | `frontend` |
| Dockerfile Path | `./Dockerfile` |
| Docker Build Context | `.` |
| Docker Command | Vazio, preservando o entrypoint da imagem |
| Health Check Path | `/health/live` |

Variáveis do frontend:

```dotenv
PORT=10000
API_PROXY_TARGET=https://URL-PUBLICA-REAL-DA-API.onrender.com
```

`API_PROXY_TARGET` deve ser uma origem HTTP(S), sem caminho `/api`, usuário/senha, query ou fragmento. Para um endereço público `onrender.com`, HTTPS é obrigatório. Nunca aponte essa variável para a própria URL do frontend.

O script da imagem gera a configuração Nginx em `/tmp` (compatível com filesystem readonly), usa o DNS do ambiente e configura Host, SNI e verificação de certificado do upstream HTTPS. No Compose, o padrão continua `http://backend:8080`; não é obrigatório definir a variável para a rede Docker local.

O navegador continua acessando apenas `https://frontend/api/...`. O proxy remove o prefixo `/api`; os cookies ficam no domínio do frontend, sem depender de cookies third-party. A API recebe o Origin original. Depois de criar o serviço frontend, copie sua URL HTTPS real para `APP_ALLOWED_ORIGINS` na API e redeploy a API.

Não use `VITE_API_URL` como atalho para chamar outro domínio: o cliente atual é same-origin. `DNS_RESOLVER`, se necessário, permite indicar um servidor DNS por IP; por padrão ele é lido do ambiente do container.

## 6. Verificação

1. API: `/health/live` e `/health/ready` devem retornar `200`.
2. Frontend: `/` mostra a apresentação pública, `/login` mostra o login e `/caixa` exige sessão.
3. Login cria cookie HttpOnly; mutações usam CSRF.
4. Administrador cadastra/edita produtos; operador não acessa a gestão.
5. Reload em `/caixa` ou `/produtos` continua funcionando pela SPA.
6. Logout invalida a sessão e uma tentativa de venda pendente continua vinculada à mesma conta.

Render termina HTTPS na borda, portanto não é necessário instalar Caddy dentro desses serviços. O Caddy do repositório é um exemplo para deploy próprio em VPS.

O ajuste de DNS/proxy é validado pelo Nginx ao iniciar. Os testes nativos não substituem o smoke test das imagens: o Docker local precisa estar disponível ou o job `images-and-e2e` do CI precisa executar. Certificados, conexão com o banco Render e domínio reais só podem ser homologados com os dados/configuração do ambiente.
