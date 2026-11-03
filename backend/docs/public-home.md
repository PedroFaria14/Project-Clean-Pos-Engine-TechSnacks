# Apresentação pública e rotas

| Rota | Conteúdo | Acesso |
| --- | --- | --- |
| `/` | Apresentação do projeto, funcionalidades, tecnologias e limites atuais | Público |
| `/login` | Login e verificação da sessão | Público; sessão válida segue para o sistema |
| `/caixa` | Frente de caixa e recuperação de tentativa pendente | Autenticado |
| `/produtos` | Gestão de produtos | Administrador |

A home não bloqueia seu conteúdo enquanto verifica sessão ou quando a API está indisponível. Não consulta catálogo nem exibe dados reais de contas/pedidos. O exemplo visual do caixa é identificado como ilustração com dados fictícios.

O conteúdo explicita que pedidos ficam aguardando pagamento. Estoque físico, confirmação de recebimento e fechamento de caixa são evoluções, não recursos anunciados como concluídos.

Uma rota protegida sem sessão leva a `/login`. Depois de entrar, apenas `/caixa` ou `/produtos` podem ser destinos; URLs externas não são aceitas. Operadores continuam sem acesso à gestão. O logotipo e os links de apresentação permitem voltar à home sem encerrar a sessão.

Tentativas de checkout continuam isoladas por usuário. A navegação não modifica o contrato de sessão, CSRF ou idempotência. Somente tentativas já iniciadas são recuperáveis; esse fluxo não promete persistência de carrinhos ainda não enviados.

Testes cobrem API lenta/indisponível, navegação, permissões, teclado, acessibilidade e viewports de 320, 768 e 1440 px. As jornadas reais usam frontend compilado, API Go e PostgreSQL isolado.

A publicação deve servir a apresentação mesmo sem API. No exemplo Caddy, o frontend é monitorado em `/health/live`; monitore `/api/health/ready` separadamente. Os Compose de desenvolvimento/produção não condicionam o frontend à prontidão do banco. A stack de testes continua aguardando os serviços para suas jornadas integradas.
