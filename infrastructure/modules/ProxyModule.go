package modules

import (
	"log"
	"strconv"

	"worker/application/dto"
	"worker/application/worker/ports"
	"worker/application/worker/usecase"
	"worker/infrastructure/adapters"
	"worker/infrastructure/controllers"
	serverhttp "worker/infrastructure/server/http"
	serversocks5 "worker/infrastructure/server/socks5"
)

// ProxyModule encapsula toda a composição do domínio proxy:
// adapters, use cases, proxy manager e controllers.
// É responsável por registrar suas rotas no HTTP server.
type ProxyModule struct {
	proxyManager ports.IProxyManager
	proxyServer  serversocks5.IProxyServer
}

// NewProxyModule compõe o módulo proxy: adapters → use cases → proxy manager → controllers.
// Registra as rotas no HTTP server.
func NewProxyModule(cfg *dto.Config, httpServer serverhttp.IServer) *ProxyModule {
	if cfg == nil {
		log.Fatal("proxy module: config nao pode ser nil")
	}
	if httpServer == nil {
		log.Fatal("proxy module: httpServer nao pode ser nil")
	}

	// === DI Interno do Módulo ===
	processAdapter := adapters.NewProcessAdapter()
	cacheAdapter := adapters.NewCacheAdapter()
	dialerBuilder := adapters.NewDialerBuilderAdapter()

	startUC := usecase.NewStartProxyUseCase(processAdapter, dialerBuilder)
	stopUC := usecase.NewStopProxyUseCase(processAdapter)
	restartUC := usecase.NewRestartProxyUseCase(processAdapter, cacheAdapter, startUC)

	proxyManager := adapters.NewProxyManager(processAdapter, startUC, stopUC, restartUC)

	bypass := adapters.NewBypassDialerWithGetter(func() ports.IContextDialer { return proxyManager.Dialer() })
	proxyAddr := ":" + strconv.Itoa(cfg.ProxyPort)
	socks5Srv := serversocks5.NewServerWithToken(proxyAddr, bypass, cfg.ProxyToken)
	proxyAdapter := serversocks5.NewServerAdapter(socks5Srv)

	// === Controllers (que registram as rotas) ===
	proxyControllers := controllers.NewProxyControllers(proxyManager, httpServer)
	proxyControllers.RegisterRoutes()

	return &ProxyModule{
		proxyManager: proxyManager,
		proxyServer:  proxyAdapter,
	}
}

// ProxyManager retorna a interface para o serviço orquestrador.
func (m *ProxyModule) ProxyManager() ports.IProxyManager {
	return m.proxyManager
}

// ProxyServer retorna o servidor SOCKS5.
func (m *ProxyModule) ProxyServer() serversocks5.IProxyServer {
	return m.proxyServer
}
