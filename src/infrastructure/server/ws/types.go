package ws

import (
	"context"

	"worker/src/domain"
)

// IClientGateway é a interface do gateway de mensagens WebSocket (transporte). Fica na infraestrutura.
type IClientGateway interface {
	Run(ctx context.Context) error
	Send(evt domain.Event) error
	Commands() <-chan domain.Command
	IsConnected() bool
	Close() error
}
