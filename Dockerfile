# Estágio de Compilação
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copia os arquivos de definição de módulo
COPY go.mod ./

# Baixa as dependências e gera o go.sum
# Como não temos go.sum localmente, o tidy vai resolver tudo
RUN go mod tidy

# Copia o código fonte
COPY . .

# Compila a aplicação
# CGO_ENABLED=0 garante um binário estático
# -ldflags="-s -w" remove informações de debug para reduzir o tamanho
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server cmd/server/main.go

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
