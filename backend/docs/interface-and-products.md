# Interface operacional e edição de produtos

## Organização do frontend

- `src/index.css`: tokens de cor, tipografia, espaçamento, estados, foco e layouts responsivos.
- `src/components/ui/`: botão, campo com label/erro, alertas, diálogo nativo, marca e estados vazios/carregando.
- `src/products/product_api.ts`: contrato tipado e validação das respostas de catálogo/edição.
- `src/products/use_products.ts`: carregamento cancelável, repetição explícita e atualização da lista após mutações.
- `src/products/money.ts`: conversão decimal para centavos e formatação monetária sem perda de precisão dentro do limite de inteiros seguros do JavaScript.
- `src/components/ProductForm.tsx`: formulário compartilhado por cadastro e edição.

Não foram adicionadas dependências visuais, fontes remotas ou bibliotecas de estado. A navegação mostra a tela ativa e mantém as permissões da sessão. O login usa a mesma linguagem visual do caixa e da gestão.

## Operação do caixa

O catálogo tem busca por nome, insensível a acentos. Essa busca é local sobre os produtos carregados; não substitui paginação ou busca no backend para um catálogo de grande volume.

O pedido tem controles de aumentar/diminuir quantidades, subtotal por item e total estimado. Cálculos que ultrapassem o limite monetário seguro são recusados antes de alterar o carrinho. O servidor continua sendo a autoridade sobre o valor final.

Sessões, CSRF, idempotência e tentativas pendentes por usuário continuam sendo usados. Uma tentativa pendente bloqueia alterações nos itens. Descartar uma tentativa exige confirmação em diálogo e não cancela uma venda no servidor.

Estados de catálogo vazio, busca sem resultado, carregamento e indisponibilidade são distintos. Falhas não são apresentadas como ausência de produtos.

## Cadastro, edição e exclusão

Administradores podem cadastrar, editar e excluir produtos. A lista é atualizada após uma resposta válida, sem exigir reload. Se uma alteração de nome não corresponder mais à busca ativa, a busca é limpa para manter o produto salvo visível.

O formulário valida nome (1 a 120 caracteres após remoção de espaços nas extremidades) e preço decimal em reais, com até duas casas decimais. Aceita `6,50` e `6.50`; não aceita separadores de milhares, expoentes ou valores negativos. Zero é permitido. A conversão usa partes inteiras/BigInt, não `parseFloat` ou multiplicação de moeda em ponto flutuante.

Durante o salvamento, os campos, envio e fechamento do diálogo ficam bloqueados. Em falhas, os campos são preservados. O ID gerado para um cadastro permanece o mesmo durante tentativas no mesmo formulário, mas **o cadastro de produtos não possui a garantia de idempotência do checkout**: após uma resposta perdida, confira o catálogo antes de repetir ou abrir um novo formulário.

A exclusão exige confirmação e só altera a lista após `204`. Falhas mantêm o diálogo e permitem nova tentativa. Excluir do catálogo não remove snapshots de pedidos.

## Contrato de edição

```http
PATCH /products/{id}
Content-Type: application/json
X-CSRF-Token: <token da sessão>
Origin: <origem permitida>

{"name":"Coxinha artesanal","price":750}
```

- Somente administrador autenticado, com Origin e CSRF válidos.
- `name` e `price` são opcionais individualmente, mas pelo menos um deve ser enviado.
- `price` é inteiro em centavos, entre zero e `9007199254740991`.
- Campos desconhecidos, `null`, corpo vazio, tipos incorretos e valores inválidos recebem `400`, com `details: [{field,message}]` ao lado de `error`, mantendo o formato de erro já usado pelo projeto.
- `200`: produto atualizado; `401`: sem sessão; `403`: sem permissão/Origin/CSRF; `404`: produto inexistente; `503`: dependência indisponível, sem detalhes internos.
- O ID não é editável. A atualização SQL altera somente os campos enviados e retorna o estado persistido.
- Atualizações simultâneas de campos diferentes não sobrescrevem campos omitidos. Para o mesmo campo, prevalece a última gravação; não há ETag/versionamento otimista neste bloco.
- O preço e nome dos pedidos anteriores e das respostas idempotentes permanecem inalterados. Novas vendas usam o catálogo atualizado.

O contrato completo está em `api/openapi.yaml`. Não há migration nova para essa edição.

Na etapa de infraestrutura, cadastro e erros antigos também foram padronizados: criação inválida retorna `400` com `details`, ID repetido retorna `409` e falhas de dependência retornam `503`. Consulte [produção e CI](production-and-ci.md) para schema, containers e suítes E2E agora versionadas.

## Acessibilidade e responsividade

Layouts adaptados para desktop, tablet e celular, com navegação por teclado, foco visível, labels associadas, mensagens por campo, regiões de status/erro e alvos de toque de pelo menos 44 px nas ações.

Diálogos usam `showModal`, nome acessível, foco inicial, contenção de Tab/Shift+Tab, Escape quando não há operação em andamento e devolução do foco ao fechar. O foco permanece no diálogo durante o processamento. Animações de carregamento respeitam `prefers-reduced-motion`.

A listagem administrativa muda sua apresentação em telas estreitas sem remover nome, preço e ações. Contraste e regras WCAG A/AA são verificados automaticamente em cenários de login, caixa, gestão e formulário; isso complementa, mas não substitui, uma avaliação completa com leitores de tela e outros navegadores.

## Validação

```powershell
# backend/
go test ./...
go vet -tags=integration ./...

# PostgreSQL descartável, com TECHSNACKS_TEST_DATABASE_URL configurada:
go test -tags=integration -run TestPostgres -v ./internal/infra/repository

# frontend/ (Node 22.18+ ou 24)
npm test
npm run lint
npm run build
```

A suíte PostgreSQL inclui permissões e CSRF do PATCH, validação, preço zero, atualização parcial concorrente e conservação do histórico/idempotência após alteração de preço. Os testes de frontend incluem limites monetários, respostas inválidas e atualização apenas dos campos alterados.
