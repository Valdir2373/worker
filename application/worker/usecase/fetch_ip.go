package usecase

import (
	"context"
	"fmt"

	"worker/application/dto"
	"worker/application/worker/ports"
)

// FetchIPUseCase obtém o IP de saída na hora (consulta externa).
type FetchIPUseCase struct {
	ipFetcher ports.IIPFetcher
}

func NewFetchIPUseCase(ipFetcher ports.IIPFetcher) *FetchIPUseCase {
	return &FetchIPUseCase{ipFetcher: ipFetcher}
}

func (uc *FetchIPUseCase) Run(ctx context.Context) (dto.IPOutput, error) {
	ip, err := uc.ipFetcher.FetchIP(ctx)
	if err != nil {
		return dto.IPOutput{}, fmt.Errorf("fetchIP: %w", err)
	}
	return dto.IPOutput{IP: ip}, nil
}
