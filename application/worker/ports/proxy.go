package ports

import "context"

// IProxyManager contrato usado pelo WorkerService e controllers.
// Start/Stop/Restart delegam para use cases; CurrentIP/FetchIP/Dialer/IsRunning usam estado dos adapters.
// Implementado por um facade em infrastructure/adapters (proxy_manager.go).
type IProxyManager interface {
	Start(ctx context.Context) error
	Stop() error
	Restart(ctx context.Context) error
	CurrentIP() string
	FetchIP(ctx context.Context) (string, error)
	Dialer() IContextDialer
	IsRunning() bool
}
