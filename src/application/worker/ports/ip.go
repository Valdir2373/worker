package ports

import "context"

// IIPFetcher obtém e armazena o IP de saída do proxy.
// Implementado por adapters na infrastructure (ex.: HTTP via TOR).
type IIPFetcher interface {
	FetchIP(ctx context.Context) (string, error)
	CurrentIP() string
	SetCurrentIP(ip string)
}
