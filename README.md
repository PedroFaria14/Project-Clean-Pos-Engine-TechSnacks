# TechSnacks POS Engine

Um sistema de Ponto de Venda (PDV) moderno e robusto desenvolvido para otimizar o atendimento e a gestão de estoque de estabelecimentos alimentícios, projetado inicialmente para as operações da Salgados. O projeto utiliza Clean Architecture para garantir um código escalável, testável e de fácil manutenção.

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


