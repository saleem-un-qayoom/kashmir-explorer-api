// Package ws — WebSocket hub for live advisory broadcasts.
//
// Pattern: clients connect to /ws/advisories, get added to the hub.
// alert-worker (or admin publish) writes an advisory then calls
// Hub.Broadcast(JSON) which fans out to every client.
//
// Backpressure: per-client send channel is buffered at 16; slow consumers
// get dropped (we log + close their conn). Total clients are unbounded —
// suitable for ~10k concurrent before we need a Redis pub-sub fan-out.
package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	pingInterval = 30 * time.Second
	writeTimeout = 10 * time.Second
	maxMessageSz = 1 << 14 // 16 KB
)

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
}

func NewHub() *Hub { return &Hub{clients: map[*Client]struct{}{}} }

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true }, // tighten in prod
}

// HandleWS — registers as an http.Handler at /ws/advisories.
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws upgrade failed", slog.Any("err", err))
		return
	}
	c := &Client{hub: h, conn: conn, send: make(chan []byte, 16)}

	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
	slog.Info("ws client connected", slog.Int("total", h.Count()))

	go c.writer()
	c.reader()
}

func (h *Hub) Count() int {
	h.mu.RLock(); defer h.mu.RUnlock()
	return len(h.clients)
}

// Broadcast a JSON-serialisable payload to every connected client.
func (h *Hub) Broadcast(payload any) {
	msg, err := json.Marshal(payload)
	if err != nil { slog.Error("ws marshal", slog.Any("err", err)); return }

	h.mu.RLock(); defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.send <- msg:
		default:
			// Slow client → drop & close.
			close(c.send)
			delete(h.clients, c)
		}
	}
}

func (h *Hub) remove(c *Client) {
	h.mu.Lock(); defer h.mu.Unlock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
}

func (c *Client) reader() {
	defer func() { c.hub.remove(c); c.conn.Close() }()
	c.conn.SetReadLimit(maxMessageSz)
	c.conn.SetReadDeadline(time.Now().Add(pingInterval * 2))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pingInterval * 2))
		return nil
	})
	for {
		// We don't expect messages from the client (advisories is one-way).
		// But we read to detect closes.
		if _, _, err := c.conn.NextReader(); err != nil { return }
	}
}

func (c *Client) writer() {
	ping := time.NewTicker(pingInterval)
	defer func() { ping.Stop(); c.conn.Close() }()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil { return }
		case <-ping.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil { return }
		}
	}
}

/* ─── Convenience: emit a snapshot to a freshly-connected client ─── */

func (h *Hub) BroadcastWithCtx(ctx context.Context, payload any) {
	select {
	case <-ctx.Done(): return
	default: h.Broadcast(payload)
	}
}
