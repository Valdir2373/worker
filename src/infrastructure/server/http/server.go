package http

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"
)

// IServer é a interface do servidor HTTP (transporte). Fica na infraestrutura, não no domínio.
type IServer interface {
	RegisterRoute(method, path string, handler interface{})
	RegisterRoutePublic(method, path string, handler interface{})
	Listen(addr string) error
	Shutdown(ctx context.Context) error
}

const headerProxyToken = "X-Proxy-Token"

type Server struct {
	mux    *http.ServeMux
	server *http.Server
	token  string // USER_PROXY_TOKEN; se vazio, não exige auth (dev)
	mu     sync.Mutex
}

func NewServer(proxyToken string) *Server {
	return &Server{
		mux:   http.NewServeMux(),
		token: proxyToken,
	}
}

func (s *Server) RegisterRoute(method, path string, handler interface{}) {
	h, ok := handler.(func(http.ResponseWriter, *http.Request))
	if !ok {
		return
	}
	wrapped := h
	if s.token != "" {
		wrapped = s.withProxyToken(h)
	}
	s.mux.HandleFunc(method+" "+path, wrapped)
}

func (s *Server) RegisterRoutePublic(method, path string, handler interface{}) {
	h, ok := handler.(func(http.ResponseWriter, *http.Request))
	if !ok {
		return
	}
	s.mux.HandleFunc(method+" "+path, h)
}

func (s *Server) withProxyToken(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(headerProxyToken) != s.token {
			log.Printf("http: token inválido ou ausente (X-Proxy-Token)")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid_or_missing_token"}`))
			return
		}
		next(w, r)
	}
}

func (s *Server) Listen(addr string) error {
	s.mu.Lock()
	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	s.mu.Unlock()
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	srv := s.server
	s.mu.Unlock()
	if srv == nil {
		return nil
	}
	return srv.Shutdown(ctx)
}

var _ IServer = (*Server)(nil)
