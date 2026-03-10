package modules

import (
	"context"
	"log"

	"worker/application/dto"
	serverhttp "worker/infrastructure/server/http"
	serverws "worker/infrastructure/server/ws"
	"worker/infrastructure/service"
)

// NewWorkerApp é uma agulha de DI: orquestra os módulos e compõe WorkerService.
// Responsabilidades:
// - Instanciar e orquestrar módulos do software
// - Passar dependências entre módulos
// - Criar o serviço de aplicação
func NewWorkerApp(cfg *dto.Config) *WorkerApp {
	if cfg == nil {
		log.Fatal("worker: config nao pode ser nil")
	}

	// === Servidores Compartilhados ===
	httpServer := serverhttp.NewServer(cfg.ProxyToken)
	wsGateway := serverws.NewServerGateway("")

	// === Módulos de Domínio ===
	proxyModule := NewProxyModule(cfg, httpServer)

	// === WebSocket Registration ===
	httpServer.RegisterRoutePublic("GET", "/ws", serverws.UpgradeHandler(wsGateway, nil, cfg.ProxyToken))

	// === Serviço de Aplicação ===
	svc := service.NewWorkerService(
		proxyModule.ProxyManager(),
		proxyModule.ProxyServer(),
		wsGateway,
		httpServer,
		cfg,
	)

	return &WorkerApp{svc: svc}
}

type WorkerApp struct {
	svc *service.WorkerService
}

func (a *WorkerApp) Run(ctx context.Context) error {
	return a.svc.Run(ctx)
}
