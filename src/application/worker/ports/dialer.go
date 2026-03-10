package ports

import (
	"context"
	"net"
)

// IContextDialer abre conexão de rede através do proxy.
// Implementado por adapters na infrastructure (ex.: SOCKS5 TOR).
type IContextDialer interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}
