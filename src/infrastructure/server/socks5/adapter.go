package socks5

import "context"

// ServerAdapter adapta Server para IProxyServer (compatibilidade com injeção).
type ServerAdapter struct {
	*Server
}

func NewServerAdapter(s *Server) *ServerAdapter {
	return &ServerAdapter{Server: s}
}

func (a *ServerAdapter) ListenAndServe(ctx context.Context) error {
	return a.Server.ListenAndServe(ctx)
}

var _ IProxyServer = (*ServerAdapter)(nil)
