package ports

// IDialerBuilder monta dialer e ipFetcher (após proxy estar disponível).
// Implementado por adapters na infrastructure (ex.: TOR SOCKS5 + HTTP client).
type IDialerBuilder interface {
	Build() (IContextDialer, IIPFetcher, error)
}
