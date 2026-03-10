package adapters

import (
	"context"
	"fmt"
	"net"
	"strings"

	"golang.org/x/net/proxy"
	"worker/src/application/worker/ports"
)

var bypassHosts = []string{"host.docker.internal", "localhost", "127.0.0.1"}

func isBypassHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	for _, h := range bypassHosts {
		if host == strings.ToLower(h) {
			return true
		}
	}
	return false
}

// BypassDialerGetter retorna o dialer do proxy atual (pode ser nil antes de Start).
type BypassDialerGetter func() ports.IContextDialer

// NewBypassDialerWithGetter retorna proxy.ContextDialer: bypass para localhost/docker.internal; resto via getter.
func NewBypassDialerWithGetter(getDialer BypassDialerGetter) proxy.ContextDialer {
	return &bypassDialerLazy{getDialer: getDialer, direct: net.Dialer{}}
}

type bypassDialerLazy struct {
	getDialer BypassDialerGetter
	direct    net.Dialer
}

func (d *bypassDialerLazy) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	if isBypassHost(host) {
		return d.direct.DialContext(ctx, network, addr)
	}
	inner := d.getDialer()
	if inner == nil {
		return nil, fmt.Errorf("dialer: proxy ainda nao iniciado")
	}
	return inner.DialContext(ctx, network, addr)
}
