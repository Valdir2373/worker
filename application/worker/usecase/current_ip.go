package usecase

import (
	"worker/application/dto"
	"worker/application/worker/ports"
)

// GetCurrentIPUseCase retorna o IP de saída armazenado (sem consulta externa).
type GetCurrentIPUseCase struct {
	ipFetcher ports.IIPFetcher
}

func NewGetCurrentIPUseCase(ipFetcher ports.IIPFetcher) *GetCurrentIPUseCase {
	return &GetCurrentIPUseCase{ipFetcher: ipFetcher}
}

func (uc *GetCurrentIPUseCase) Run() dto.IPOutput {
	return dto.IPOutput{IP: uc.ipFetcher.CurrentIP()}
}
