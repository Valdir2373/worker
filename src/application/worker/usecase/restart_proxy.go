package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

	"worker/src/application/worker/ports"
)

const restartDelay = 10 * time.Second

// RestartProxyUseCase para o proxy, limpa cache e sobe de novo (novo circuito).
type RestartProxyUseCase struct {
	process  ports.IProcessController
	cache    ports.ICacheCleaner
	startUC  *StartProxyUseCase
}

func NewRestartProxyUseCase(
	process ports.IProcessController,
	cache ports.ICacheCleaner,
	startUC *StartProxyUseCase,
) *RestartProxyUseCase {
	return &RestartProxyUseCase{process: process, cache: cache, startUC: startUC}
}

// Run retorna o novo dialer e ipFetcher após o restart.
func (uc *RestartProxyUseCase) Run(ctx context.Context) (dialer ports.IContextDialer, ipFetcher ports.IIPFetcher, err error) {
	if err := uc.process.Stop(); err != nil {
		return nil, nil, fmt.Errorf("restart/stop: %w", err)
	}
	log.Printf("proxy: aguardando %v...", restartDelay)
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case <-time.After(restartDelay):
	}
	uc.cache.CleanCache()
	log.Printf("proxy: iniciando após restart...")
	return uc.startUC.Run(ctx)
}
