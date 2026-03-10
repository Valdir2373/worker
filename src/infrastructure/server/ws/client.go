package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"worker/domain"
)

var backoffDurations = []time.Duration{
	1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second,
	16 * time.Second, 32 * time.Second, 60 * time.Second,
}

const cmdChanBuf = 16
const logConnectThrottle = 3
const logConnectInterval = 60 * time.Second

type Client struct {
	url      string
	workerID string
	mu       sync.Mutex
	conn     *websocket.Conn
	cmdCh    chan domain.Command
	done     chan struct{}
	connected atomic.Bool
}

func NewClient(url, workerID string) *Client {
	return &Client{
		url: url, workerID: workerID,
		cmdCh: make(chan domain.Command, cmdChanBuf),
		done:  make(chan struct{}),
	}
}

func (c *Client) Run(ctx context.Context) error {
	attempt := 0
	var lastLog time.Time
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.done:
			return nil
		default:
		}
		conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.url, nil)
		if err != nil {
			wait := backoffDurations[min(attempt, len(backoffDurations)-1)]
			if attempt < logConnectThrottle || time.Since(lastLog) >= logConnectInterval {
				log.Printf("ws: indisponível (%s) — tentativa em %v", err, wait)
				lastLog = time.Now()
			}
			if !c.sleep(ctx, wait) {
				return ctx.Err()
			}
			attempt++
			continue
		}
		log.Printf("ws: conectado em %s", c.url)
		c.mu.Lock()
		c.conn = conn
		c.mu.Unlock()
		c.connected.Store(true)
		attempt = 0
		c.readPump(conn)
		c.connected.Store(false)
		c.mu.Lock()
		c.conn = nil
		c.mu.Unlock()
		select {
		case <-c.done:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		wait := backoffDurations[min(attempt, len(backoffDurations)-1)]
		if !c.sleep(ctx, wait) {
			return ctx.Err()
		}
		attempt++
	}
}

func (c *Client) Send(evt domain.Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("ws: não conectado")
	}
	b, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(websocket.TextMessage, b)
}

func (c *Client) Commands() <-chan domain.Command {
	return c.cmdCh
}

func (c *Client) IsConnected() bool {
	return c.connected.Load()
}

func (c *Client) Close() error {
	select {
	case <-c.done:
	default:
		close(c.done)
	}
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn != nil {
		return conn.Close()
	}
	return nil
}

func (c *Client) readPump(conn *websocket.Conn) {
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
		case c.cmdCh <- cmd:
		default:
		}
	}
}

func (c *Client) sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	case <-c.done:
		return false
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var _ IClientGateway = (*Client)(nil)
