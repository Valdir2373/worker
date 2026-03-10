package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

	"worker/application/worker/ports"
)

const (
	pollIPTimeout  = 120 * time.Second
	pollIPInterval = 2 * time.Second
)

// StartProxyUseCase sobe o proxy, espera estar pronto e descobre o IP de saída.
type StartProxyUseCase struct {
	process ports.IProcessController
	builder ports.IDialerBuilder
}

func NewStartProxyUseCase(process ports.IProcessController, builder ports.IDialerBuilder) *StartProxyUseCase {
	return &StartProxyUseCase{process: process, builder: builder}
}

// Run retorna o dialer e o ipFetcher prontos para uso. O facade deve guardá-los.
func (uc *StartProxyUseCase) Run(ctx context.Context) (dialer ports.IContextDialer, ipFetcher ports.IIPFetcher, err error) {
	dialer, ipFetcher, err = uc.builder.Build()
	if err != nil {
		return nil, nil, fmt.Errorf("builder: %w", err)
	}
	if !uc.process.IsRunning() {
		if err := uc.process.StartProcess(); err != nil {
			return nil, nil, fmt.Errorf("startProcess: %w", err)
		}
	}
	if err := uc.process.WaitReady(ctx); err != nil {
		return nil, nil, fmt.Errorf("waitReady: %w", err)
	}
	ip, err := uc.pollIP(ctx, ipFetcher)
	if err != nil {
		return nil, nil, fmt.Errorf("pollIP: %w", err)
	}
	ipFetcher.SetCurrentIP(ip)
	log.Printf("proxy: pronto — IP de saída: %s", ip)
	return dialer, ipFetcher, nil
}

func (uc *StartProxyUseCase) pollIP(ctx context.Context, ipFetcher ports.IIPFetcher) (string, error) {
	log.Printf("proxy: aguardando IP...")
	deadline := time.Now().Add(pollIPTimeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		ip, err := ipFetcher.FetchIP(ctx)
		if err == nil && ip != "" {
			return ip, nil
		}
		if err != nil {
			log.Printf("proxy: polling IP falhou: %v", err)
		}
		time.Sleep(pollIPInterval)
	}
	return "", fmt.Errorf("timeout aguardando IP (%v)", pollIPTimeout)
}
