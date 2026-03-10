package ports

import "context"

// IProcessController controla o processo do proxy (subir, descer, esperar pronto).
// Implementado por adapters na infrastructure (ex.: processo TOR).
type IProcessController interface {
	StartProcess() error
	Stop() error
	WaitReady(ctx context.Context) error
	IsRunning() bool
}
