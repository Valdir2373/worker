# ─── Stage 1: build ───────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
COPY . .
RUN ls -laR .

# ─── Stage 2: runtime ─────────────────────────────────────────────────────────
FROM alpine:latest

RUN apk add --no-cache tor ca-certificates su-exec netcat-openbsd \
    && mkdir -p /var/lib/tor \
    && chown -R tor:tor /var/lib/tor \
    && chmod 700 /var/lib/tor

WORKDIR /app

COPY --from=build /worker .
COPY torrc /etc/tor/torrc
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

EXPOSE 8080
EXPOSE 1080

ENTRYPOINT ["./entrypoint.sh"]
