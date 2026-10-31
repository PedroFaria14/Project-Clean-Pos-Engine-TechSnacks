# TechSnacks POS Engine

Um sistema de Ponto de Venda (PDV) em evolução, desenvolvido para organizar o atendimento e a gestão de catálogo de estabelecimentos alimentícios, projetado inicialmente para as operações da Salgados. O backend utiliza camadas de Clean Architecture para separar regras de negócio, casos de uso e infraestrutura.

##  Tecnologias Utilizadas

**Backend:**
* Go (Golang)
* Gin Web Framework (Roteamento e API REST)
* GORM (ORM)
* PostgreSQL (Banco de Dados Relacional)

**Frontend:**
* React (com Vite)
* TypeScript
* React Router DOM

**Infraestrutura:**
* Docker e Docker Compose

##  Arquitetura (Clean Architecture)

O backend em Go foi estritamente dividido em camadas para isolar regras de negócio das tecnologias externas:

* **Domain:** Entidades centrais do negócio (Produto, Pedido, Preço em centavos).
* **Application (Use Cases):** Casos de uso do sistema (Criar Produto, Checkout, Listar, Excluir).
* **Port:** Interfaces que definem os contratos dos repositórios.
* **Infra:** Implementações técnicas (GORM, PostgreSQL) e roteadores (Gin).

## Funcionalidades

* **Frente de Caixa (PDV):** Interface em tempo real para adição de produtos ao carrinho, cálculo automático de totais e finalização de vendas.
* **Gestão de Produtos:** Cadastro de novos itens no cardápio, listagem atualizada e exclusão direta do banco de dados.
* **Integração Fluida:** API RESTful conectada ao React com tratamento nativo de CORS.

## Checkout: schema e recuperação

Antes de iniciar a nova API, aplique explicitamente a migration de pedidos no banco configurado. O startup verifica a versão do schema e não aplica migrations automaticamente.

Consulte [estados, idempotência, migrations e testes](backend/docs/checkout-persistence.md) e o [contrato OpenAPI](backend/api/openapi.yaml).

## Login e permissões

As rotas de negócio exigem autenticação. Antes do primeiro uso, aplique as migrations e crie uma conta administrativa com o comando documentado em [autenticação e operação](backend/docs/authentication.md). Não existe usuário ou senha padrão.

Operadores acessam o caixa e os próprios pedidos; administradores também gerenciam produtos e consultam os demais pedidos. Em desenvolvimento, o frontend usa `/api` com proxy Vite para a API; em produção, configure HTTPS, origens permitidas e reverse proxy conforme o guia.

## Interface e gestão de produtos

O caixa tem busca de produtos, controles de quantidade e recuperação de tentativas pendentes por conta. Administradores têm cadastro, edição de nome/preço e exclusão com confirmação, com atualização da lista após salvar. A interface usa componentes compartilhados, layouts responsivos e feedback acessível.

Veja [interface operacional, edição de produtos e validação](backend/docs/interface-and-products.md).

## Infraestrutura e CI

Builds multi-stage, probes, Compose por ambiente, suíte E2E versionada e pipeline estão descritos no [guia de produção e CI](backend/docs/production-and-ci.md). A API exige a migration do catálogo; o CI inclui testes reais, scans, backup/restore sintético e artifacts, sem deploy automático.


