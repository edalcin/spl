# Estágio de Compilação
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copia os arquivos de definição de módulo
COPY go.mod ./

# Copia o código fonte
COPY . .

# Instala git (necessário para algumas dependências) e ajusta dependências
# Adicionando gcc e musl-dev para garantir compatibilidade de build
RUN apk add --no-cache git gcc musl-dev && go mod tidy

# Compila a aplicação
# CGO_ENABLED=0 garante um binário estático
# -ldflags="-s -w" remove informações de debug para reduzir o tamanho
RUN CGO_ENABLED=0 GOOS=linux go build -v -ldflags="-s -w" -o server ./cmd/server

# Estágio Final
FROM alpine:latest

WORKDIR /root/

# Copia o binário do estágio de build
COPY --from=builder /app/server .

# Define a porta e o volume para o banco de dados
ENV PORT=8080
ENV DB_PATH=/data/shopping.db

# Cria o diretório para o volume
RUN mkdir /data

# Expõe a porta
EXPOSE 8080

# Volume para persistência
VOLUME ["/data"]

# Comando de execução
CMD ["./server"]
