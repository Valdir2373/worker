package socks5

import "context"

// IProxyServer é a interface do servidor proxy SOCKS5 (infraestrutura). Fica no módulo server/socks5.
type IProxyServer interface {
	ListenAndServe(ctx context.Context) error
}
