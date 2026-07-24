FROM golang:1.25-alpine

WORKDIR /app

# Instala dependências essenciais do sistema
RUN apk add --no-cache gcc musl-dev

# Copia os gerenciadores de dependência do Go primeiro
COPY go.mod go.sum ./
RUN go mod download

# Copia TODO o seu código para dentro do contêiner
COPY . .

# Agora sim, roda o caminho correto!
CMD ["go", "run", "cmd/api/main.go"]