#!/bin/sh
set -e

# Em deploy (Render etc.) o usuário não define TOR_CONTROL_PASSWORD; geramos aqui.
# A senha protege o ControlPort (127.0.0.1:9051): só o processo worker controla o TOR.
if [ -z "$TOR_CONTROL_PASSWORD" ]; then
    TOR_CONTROL_PASSWORD=$(od -An -N16 -tx1 /dev/urandom | tr -d ' \n')
    export TOR_CONTROL_PASSWORD
    echo "[entrypoint] TOR_CONTROL_PASSWORD gerado automaticamente"
fi

echo "[entrypoint] gerando hash da senha de controle..."
HASH=$(tor --hash-password "$TOR_CONTROL_PASSWORD" 2>/dev/null | grep "^16:")
if [ -z "$HASH" ]; then
    echo "[entrypoint] ERRO: falha ao gerar hash"
    exit 1
fi

sed -i "s|HashedControlPassword PLACEHOLDER|HashedControlPassword ${HASH}|" /etc/tor/torrc
echo "[entrypoint] hash configurado no torrc"

echo "[entrypoint] iniciando TOR como usuário tor..."
su-exec tor tor -f /etc/tor/torrc &
TOR_PID=$!

echo "[entrypoint] aguardando ControlPort 9051..."
MAX_WAIT=60
ELAPSED=0
until nc -w 1 127.0.0.1 9051 </dev/null 2>/dev/null; do
    if [ $ELAPSED -ge $MAX_WAIT ]; then
        echo "[entrypoint] ERRO: TOR não inicializou em ${MAX_WAIT}s"
        kill "$TOR_PID" 2>/dev/null
        exit 1
    fi
    if ! kill -0 "$TOR_PID" 2>/dev/null; then
        echo "[entrypoint] ERRO: Processo TOR morreu prematuramente"
        exit 1
    fi
    sleep 1
    ELAPSED=$((ELAPSED + 1))
done

echo "[entrypoint] TOR pronto (PID $TOR_PID)"
echo "[entrypoint] iniciando worker..."

exec ./worker
