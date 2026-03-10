package adapters

import (
	"context"
	"sync"

	"worker/src/application/worker/ports"
	"worker/src/application/worker/usecase"
)

// ProxyManager implementa ports.IProxyManager: delega Start/Stop/Restart para use cases; CurrentIP/FetchIP/Dialer/IsRunning para estado.
type ProxyManager struct {
	process  ports.IProcessController
	startUC  *usecase.StartProxyUseCase
	stopUC   *usecase.StopProxyUseCase
	restartUC *usecase.RestartProxyUseCase

	mu               sync.RWMutex
	currentDialer    ports.IContextDialer
	currentIPFetcher ports.IIPFetcher
}

func NewProxyManager(
	process ports.IProcessController,
	startUC *usecase.StartProxyUseCase,
	stopUC *usecase.StopProxyUseCase,
	restartUC *usecase.RestartProxyUseCase,
) *ProxyManager {
	return &ProxyManager{
		process:   process,
		startUC:   startUC,
		stopUC:    stopUC,
		restartUC: restartUC,
	}
}

func (m *ProxyManager) Start(ctx context.Context) error {
	dialer, ipFetcher, err := m.startUC.Run(ctx)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.currentDialer = dialer
	m.currentIPFetcher = ipFetcher
	m.mu.Unlock()
	return nil
}

func (m *ProxyManager) Stop() error {
	return m.stopUC.Run()
}

func (m *ProxyManager) Restart(ctx context.Context) error {
	dialer, ipFetcher, err := m.restartUC.Run(ctx)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.currentDialer = dialer
	m.currentIPFetcher = ipFetcher
	m.mu.Unlock()
	return nil
}

func (m *ProxyManager) CurrentIP() string {
	m.mu.RLock()
	ipf := m.currentIPFetcher
	m.mu.RUnlock()
	if ipf == nil {
		return ""
	}
	return ipf.CurrentIP()
}

func (m *ProxyManager) FetchIP(ctx context.Context) (string, error) {
	m.mu.RLock()
	ipf := m.currentIPFetcher
	m.mu.RUnlock()
	if ipf == nil {
		return "", nil
	}
	return ipf.FetchIP(ctx)
}

func (m *ProxyManager) Dialer() ports.IContextDialer {
	m.mu.RLock()
	d := m.currentDialer
	m.mu.RUnlock()
	return d
}

func (m *ProxyManager) IsRunning() bool {
	return m.process.IsRunning()
}

var _ ports.IProxyManager = (*ProxyManager)(nil)
