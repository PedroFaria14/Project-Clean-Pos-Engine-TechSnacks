FROM node:20-alpine

WORKDIR /app

# Instala o pnpm globalmente dentro do contêiner
RUN npm install -g pnpm

# Copia os arquivos de dependência primeiro (para usar o cache do Docker)
COPY package.json pnpm-lock.yaml* ./
RUN pnpm install

# Copia o resto do código
COPY . .

EXPOSE 5173

# Inicia o servidor do Vite
CMD ["pnpm", "dev"]