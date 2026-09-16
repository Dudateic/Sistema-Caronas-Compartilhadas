FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# Compila os executáveis
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/servidor ./aplicacao/servidor/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/motorista ./aplicacao/motorista/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/passageiro ./aplicacao/passageiro/main.go

# Imagem
FROM alpine:latest

LABEL authors="duda"

COPY --from=builder /bin/servidor /usr/local/bin/servidor
COPY --from=builder /bin/motorista /usr/local/bin/motorista
COPY --from=builder /bin/passageiro /usr/local/bin/passageiro

WORKDIR /app

EXPOSE 8081

CMD ["servidor"]