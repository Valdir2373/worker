package ws

import (
	"context"

	"worker/domain"
)

type NoOpClient struct {
	cmdCh chan domain.Command
}

func NewNoOpClient() *NoOpClient {
	return &NoOpClient{cmdCh: make(chan domain.Command)}
}

func (n *NoOpClient) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func (n *NoOpClient) Send(evt domain.Event) error {
	return nil
}

func (n *NoOpClient) Commands() <-chan domain.Command {
	return n.cmdCh
}

func (n *NoOpClient) IsConnected() bool {
	return false
}

func (n *NoOpClient) Close() error {
	return nil
}

var _ IClientGateway = (*NoOpClient)(nil)
