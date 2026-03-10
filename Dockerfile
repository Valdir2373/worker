# ─── Stage 1: build ───────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS build
WORKDIR /app

# Copia arquivos de dependências primeiro (otimiza cache)
COPY go.mod go.sum ./
RUN go mod download

# Copia todo o código fonte
COPY . .

# --- DEPURAÇÃO: Lista tudo antes do build para validar os caminhos ---
RUN echo "Estrutura de arquivos no container de build:" && ls -laR .

# Executa o build (onde o erro ocorria)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /worker ./src/main.go

# ─── Stage 2: runtime ─────────────────────────────────────────────────────────
FROM alpine:latest

RUN apk add --no-cache tor ca-certificates su-exec netcat-openbsd \
    && mkdir -p /var/lib/tor \
    && chown -R tor:tor /var/lib/tor \
    && chmod 700 /var/lib/tor

WORKDIR /app

# Copia o binário gerado
COPY --from=build /worker .
COPY torrc /etc/tor/torrc
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

# --- DEPURAÇÃO: Lista o diretório final ---
RUN echo "Arquivos no diretório de runtime:" && ls -la .

EXPOSE 8080
EXPOSE 1080

ENTRYPOINT ["./entrypoint.sh"]