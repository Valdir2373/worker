package service

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"time"

	"worker/application/dto"
	"worker/application/worker/ports"
	"worker/domain"
	serverhttp "worker/infrastructure/server/http"
	serversocks5 "worker/infrastructure/server/socks5"
	serverws "worker/infrastructure/server/ws"
)

// WorkerService orquestra o ciclo de vida do Worker: proxy (IProxyManager), SOCKS5, HTTP e WebSocket.
type WorkerService struct {
	proxyMgr ports.IProxyManager
	proxy    serversocks5.IProxyServer
	ws       serverws.IClientGateway
	server   serverhttp.IServer
	cfg      *dto.Config
}

func NewWorkerService(
	proxyMgr ports.IProxyManager,
	proxy serversocks5.IProxyServer,
	ws serverws.IClientGateway,
	server serverhttp.IServer,
	cfg *dto.Config,
) *WorkerService {
	return &WorkerService{proxyMgr: proxyMgr, proxy: proxy, ws: ws, server: server, cfg: cfg}
}

func (s *WorkerService) Run(ctx context.Context) error {
	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()
	startTime := time.Now()

	log.Printf("worker: iniciando proxy bootstrap...")
	if err := s.proxyMgr.Start(runCtx); err != nil {
		return fmt.Errorf("worker: proxy bootstrap falhou: %w", err)
	}
	log.Printf("worker: proxy pronto — IP: %s", s.proxyMgr.CurrentIP())

	go func() {
		if err := s.proxy.ListenAndServe(runCtx); err != nil {
			log.Printf("worker: socks5 encerrado: %v", err)
		}
	}()

	listenAddr := ":" + strconv.Itoa(s.cfg.ListenPort) // cfg já validado no config.Load()
	go func() {
		log.Printf("worker: HTTP server em %s", listenAddr)
		if err := s.server.Listen(listenAddr); err != nil && err != context.Canceled {
			log.Printf("worker: HTTP server erro: %v", err)
		}
	}()

	go s.ws.Run(runCtx) //nolint:errcheck
	s.waitConnected(runCtx, 5*time.Second)
	initialConnected := s.ws.IsConnected()
	if initialConnected {
		s.sendRegisterAndIPReady()
	}
	go s.reconnectMonitor(runCtx, initialConnected)

	s.commandLoop(runCtx, cancelRun, startTime)
	s.shutdown(runCtx)
	return nil
}

func (s *WorkerService) commandLoop(ctx context.Context, cancelRun context.CancelFunc, startTime time.Time) {
	cmdCh := s.ws.Commands()
	var refreshC <-chan time.Time
	var refreshTimer *time.Timer
	if s.cfg.AutoRefreshMinutes > 0 {
		d := time.Duration(s.cfg.AutoRefreshMinutes) * time.Minute
		refreshTimer = time.NewTimer(d)
		refreshC = refreshTimer.C
		defer refreshTimer.Stop()
	}
	resetRefresh := func() {
		if refreshTimer == nil {
			return
		}
		if !refreshTimer.Stop() {
			select { case <-refreshTimer.C: default: }
		}
		refreshTimer.Reset(time.Duration(s.cfg.AutoRefreshMinutes) * time.Minute)
	}
	for {
		select {
		case cmd := <-cmdCh:
			s.handleCommand(ctx, cmd, cancelRun, resetRefresh, startTime)
		case <-refreshC:
			log.Printf("worker: auto-refresh (intervalo: %d min)", s.cfg.AutoRefreshMinutes)
			s.regenerate(ctx)
			resetRefresh()
		case <-ctx.Done():
			return
		}
	}
}

func (s *WorkerService) handleCommand(ctx context.Context, cmd domain.Command, cancelRun context.CancelFunc, resetRefresh func(), startTime time.Time) {
	switch cmd.Command {
	case "regenerate":
		s.regenerate(ctx)
		resetRefresh()
	case "ping":
		_ = s.ws.Send(domain.Event{
			Event: "pong", IP: s.proxyMgr.CurrentIP(),
			UptimeSeconds: int64(time.Since(startTime).Seconds()), Goroutines: runtime.NumGoroutine(),
		})
	case "shutdown":
		log.Printf("worker: shutdown recebido via WS")
		cancelRun()
	default:
		log.Printf("worker: comando desconhecido: %q", cmd.Command)
	}
}

func (s *WorkerService) regenerate(ctx context.Context) {
	log.Printf("worker: regenerando IP...")
	_ = s.ws.Send(domain.Event{Event: "ip_generating"})
	if err := s.proxyMgr.Restart(ctx); err != nil {
		log.Printf("worker: Restart falhou: %v", err)
		_ = s.ws.Send(domain.Event{Event: "error", Message: err.Error()})
		return
	}
	ip := s.proxyMgr.CurrentIP()
	log.Printf("worker: novo IP: %s", ip)
	_ = s.ws.Send(domain.Event{Event: "ip_ready", IP: ip})
}

func (s *WorkerService) sendRegisterAndIPReady() {
	_ = s.ws.Send(domain.Event{Event: "register", Socks5Port: s.cfg.ProxyPort, ListenPort: s.cfg.ListenPort})
	_ = s.ws.Send(domain.Event{Event: "ip_ready", IP: s.proxyMgr.CurrentIP()})
}

func (s *WorkerService) reconnectMonitor(ctx context.Context, initialConnected bool) {
	prev := initialConnected
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			now := s.ws.IsConnected()
			if now && !prev {
				log.Printf("worker: WS reconectado — reenviando register e ip_ready")
				s.sendRegisterAndIPReady()
			}
			prev = now
		case <-ctx.Done():
			return
		}
	}
}

func (s *WorkerService) waitConnected(ctx context.Context, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if s.ws.IsConnected() {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (s *WorkerService) shutdown(ctx context.Context) {
	log.Printf("worker: iniciando graceful shutdown...")
	_ = s.ws.Close()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = s.server.Shutdown(shutdownCtx)
	_ = s.proxyMgr.Stop()
	log.Printf("worker: encerrado")
}
