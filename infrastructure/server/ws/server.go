package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"worker/domain"
)

var defaultUpgrader = &websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type ServerGateway struct {
	workerID string
	mu       sync.Mutex
	conn     *websocket.Conn
	cmdCh    chan domain.Command
	done     chan struct{}
	connected atomic.Bool
	connDone chan struct{}
}

func NewServerGateway(workerID string) *ServerGateway {
	return &ServerGateway{
		workerID: workerID,
		cmdCh:    make(chan domain.Command, cmdChanBuf),
		done:     make(chan struct{}),
	}
}

func (g *ServerGateway) Run(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-g.done:
		return nil
	}
}

func (g *ServerGateway) Accept(conn *websocket.Conn) {
	g.mu.Lock()
	if g.conn != nil {
		_ = g.conn.Close()
		g.conn = nil
	}
	g.conn = conn
	g.connDone = make(chan struct{})
	g.mu.Unlock()
	g.connected.Store(true)
	log.Printf("ws: orquestrador conectado")

	defer func() {
		g.connected.Store(false)
		g.mu.Lock()
		g.conn = nil
		g.mu.Unlock()
		log.Printf("ws: orquestrador desconectado")
	}()

	go g.readPump(conn)
	<-g.connDone
}

func (g *ServerGateway) readPump(conn *websocket.Conn) {
	defer func() {
		if g.connDone != nil {
			close(g.connDone)
		}
	}()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var cmd domain.Command
		if json.Unmarshal(msg, &cmd) != nil || cmd.Command == "" {
			continue
		}
		select {
		case g.cmdCh <- cmd:
		default:
		}
	}
}

func (g *ServerGateway) Send(evt domain.Event) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.conn == nil {
		return fmt.Errorf("ws: nenhum orquestrador conectado")
	}
	b, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	return g.conn.WriteMessage(websocket.TextMessage, b)
}

func (g *ServerGateway) Commands() <-chan domain.Command {
	return g.cmdCh
}

func (g *ServerGateway) IsConnected() bool {
	return g.connected.Load()
}

func (g *ServerGateway) Close() error {
	select {
	case <-g.done:
	default:
		close(g.done)
	}
	g.mu.Lock()
	conn := g.conn
	g.mu.Unlock()
	if conn != nil {
		return conn.Close()
	}
	return nil
}

func UpgradeHandler(gateway *ServerGateway, upgrader *websocket.Upgrader, token string) func(http.ResponseWriter, *http.Request) {
	if upgrader == nil {
		upgrader = defaultUpgrader
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if token != "" {
			q := r.URL.Query().Get("token")
			if q != token {
				log.Printf("ws: token inválido ou ausente (query ?token=)")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"invalid_or_missing_token"}`))
				return
			}
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("ws: upgrade falhou: %v", err)
			return
		}
		gateway.Accept(conn)
	}
}

var _ IClientGateway = (*ServerGateway)(nil)
