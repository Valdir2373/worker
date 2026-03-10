package usecase

import (
	"fmt"

	"worker/application/worker/ports"
)

// StopProxyUseCase encerra o processo do proxy.
type StopProxyUseCase struct {
	process ports.IProcessController
}

func NewStopProxyUseCase(process ports.IProcessController) *StopProxyUseCase {
	return &StopProxyUseCase{process: process}
}

func (uc *StopProxyUseCase) Run() error {
	if err := uc.process.Stop(); err != nil {
		return fmt.Errorf("stop: %w", err)
	}
	return nil
}
