package controllers

import (
	"log"
	"time"

	"worker/src/application/worker/ports"
	serverhttp "worker/src/infrastructure/server/http"
)

// ProxyControllers encapsula os controllers do proxy e registra suas rotas.
type ProxyControllers struct {
	proxyMgr   ports.IProxyManager
	httpServer serverhttp.IServer
	startTime  time.Time
}

// NewProxyControllers instancia o controller do proxy.
func NewProxyControllers(proxyMgr ports.IProxyManager, httpServer serverhttp.IServer) *ProxyControllers {
	return &ProxyControllers{
		proxyMgr:   proxyMgr,
		httpServer: httpServer,
		startTime:  time.Now(),
	}
}

// RegisterRoutes registra as rotas do módulo proxy no HTTP server.
func (c *ProxyControllers) RegisterRoutes() {
	handlers := serverhttp.NewHandlers(c.proxyMgr, c.startTime)

	c.httpServer.RegisterRoutePublic("GET", "/health", handlers.HandleHealth)
	log.Printf("proxy: rota registrada — GET /health")

	c.httpServer.RegisterRoute("GET", "/getIp", handlers.HandleGetIP)
	log.Printf("proxy: rota registrada — GET /getIp")
}
