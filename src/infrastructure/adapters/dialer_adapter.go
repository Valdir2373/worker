package adapters

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/proxy"
	"worker/src/application/worker/ports"
)

const (
	torSocksAddr         = "127.0.0.1:9050"
	proxyTLSTimeout      = 30 * time.Second
	proxyResponseTimeout = 45 * time.Second
	proxyClientTimeout   = 90 * time.Second
)

// contextDialerWrap adapts proxy.ContextDialer to ports.IContextDialer.
type contextDialerWrap struct {
	d proxy.ContextDialer
}

func (w *contextDialerWrap) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return w.d.DialContext(ctx, network, addr)
}

type dnsLeakGuardDialer struct {
	inner proxy.ContextDialer
}

func (d *dnsLeakGuardDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		if ip := net.ParseIP(host); ip != nil && !ip.IsLoopback() && !ip.IsPrivate() {
			log.Printf("WARNING: possivel DNS leak - dial por IP %q", addr)
		}
	}
	return d.inner.DialContext(ctx, network, addr)
}

type dialerWrapper struct {
	fn func(context.Context, string, string) (net.Conn, error)
}

func (d *dialerWrapper) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return d.fn(ctx, network, addr)
}

func newTorDialer() (proxy.ContextDialer, error) {
	d, err := proxy.SOCKS5("tcp", torSocksAddr, nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("dialer: %w", err)
	}
	var cd proxy.ContextDialer
	if nativeCD, ok := d.(proxy.ContextDialer); ok {
		cd = nativeCD
	} else {
		cd = &dialerWrapper{contextAwareDialer(d)}
	}
	return &dnsLeakGuardDialer{inner: cd}, nil
}

func contextAwareDialer(d proxy.Dialer) func(context.Context, string, string) (net.Conn, error) {
	if cd, ok := d.(proxy.ContextDialer); ok {
		return cd.DialContext
	}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		type result struct {
			conn net.Conn
			err  error
		}
		ch := make(chan result, 1)
		go func() {
			conn, err := d.Dial(network, addr)
			ch <- result{conn, err}
		}()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case r := <-ch:
			return r.conn, r.err
		}
	}
}

func newTorTransport(d proxy.ContextDialer) *http.Transport {
	return &http.Transport{
		DialContext:           d.DialContext,
		TLSHandshakeTimeout:   proxyTLSTimeout,
		ResponseHeaderTimeout: proxyResponseTimeout,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     false,
	}
}

func newTorClient(t *http.Transport) *http.Client {
	return &http.Client{
		Transport: t,
		Timeout:   proxyClientTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("muitos redirecionamentos")
			}
			return nil
		},
	}
}

// DialerBuilderAdapter implements ports.IDialerBuilder (TOR SOCKS5 + HTTP client).
type DialerBuilderAdapter struct{}

func NewDialerBuilderAdapter() *DialerBuilderAdapter {
	return &DialerBuilderAdapter{}
}

func (b *DialerBuilderAdapter) Build() (ports.IContextDialer, ports.IIPFetcher, error) {
	d, err := newTorDialer()
	if err != nil {
		return nil, nil, fmt.Errorf("dialer: %w", err)
	}
	t := newTorTransport(d)
	c := newTorClient(t)
	ipFetcher := NewIPFetcherAdapter(c)
	return &contextDialerWrap{d: d}, ipFetcher, nil
}

var _ ports.IDialerBuilder = (*DialerBuilderAdapter)(nil)
