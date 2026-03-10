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

echo "[entrypoint] verificando permissões de /var/lib/tor..."
ls -la /var/lib/tor 2>&1 || echo "[entrypoint] NOT FOUND: /var/lib/tor"

echo "[entrypoint] iniciando TOR como root..."
tor -f /etc/tor/torrc > /tmp/tor.log 2>&1 &
TOR_PID=$!
echo "[entrypoint] TOR PID: $TOR_PID"

echo "[entrypoint] aguardando ControlPort 9051..."
MAX_WAIT=30
ELAPSED=0
until nc -z 127.0.0.1 9051 2>/dev/null; do
    if [ $ELAPSED -ge $MAX_WAIT ]; then
        echo "[entrypoint] ERRO: TOR não inicializou em ${MAX_WAIT}s"
        echo "[entrypoint] últimas linhas do log TOR:"
        tail -20 /tmp/tor.log 2>/dev/null || echo "(sem log disponível)"
        kill "$TOR_PID" 2>/dev/null || true
        exit 1
    fi
    if ! kill -0 "$TOR_PID" 2>/dev/null; then
        echo "[entrypoint] ERRO: Processo TOR morreu prematuramente"
        echo "[entrypoint] log do TOR:"
        cat /tmp/tor.log 2>/dev/null || echo "(sem log disponível)"
        exit 1
    fi
    ELAPSED=$((ELAPSED + 1))
    sleep 1
done

echo "[entrypoint] TOR pronto (PID $TOR_PID)"
echo "[entrypoint] iniciando worker..."

exec ./worker