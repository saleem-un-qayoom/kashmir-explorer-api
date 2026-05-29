// Rooms — extension to the WebSocket hub for room-based broadcasts.
//
// /ws/group/{code}    — trip group: small (2-12 people), share own GPS
// /ws/crowd/{slug}    — trek crowd: open, anonymous read-only density updates
// /ws/advisories      — global advisories (legacy single hub)
package ws

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

type room struct {
	mu      sync.RWMutex
	members map[*Client]struct{}
}

type Rooms struct {
	mu sync.RWMutex
	by map[string]*room
}

func NewRooms() *Rooms { return &Rooms{by: map[string]*room{}} }

func (r *Rooms) HandleGroup(w http.ResponseWriter, req *http.Request) {
	code := chi.URLParam(req, "code")
	r.handle(w, req, "group:"+code)
}
func (r *Rooms) HandleCrowd(w http.ResponseWriter, req *http.Request) {
	slug := chi.URLParam(req, "slug")
	r.handle(w, req, "crowd:"+slug)
}

// BroadcastRoom sends a JSON payload to every client in the named room.
func (r *Rooms) BroadcastRoom(name string, payload any) {
	r.mu.RLock()
	rm := r.by[name]
	r.mu.RUnlock()
	if rm == nil {
		return
	}
	msg, _ := json.Marshal(payload)
	rm.mu.RLock()
	for c := range rm.members {
		select {
		case c.send <- msg:
		default:
			// drop slow client
		}
	}
	rm.mu.RUnlock()
}

func (r *Rooms) handle(w http.ResponseWriter, req *http.Request, key string) {
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		slog.Error("ws upgrade", slog.Any("err", err))
		return
	}

	c := &Client{conn: conn, send: make(chan []byte, 16)}
	r.mu.Lock()
	rm, ok := r.by[key]
	if !ok {
		rm = &room{members: map[*Client]struct{}{}}
		r.by[key] = rm
	}
	r.mu.Unlock()
	rm.mu.Lock()
	rm.members[c] = struct{}{}
	rm.mu.Unlock()

	slog.Info("ws joined room", slog.String("room", key))

	go r.writer(c)
	r.reader(c, key, rm)
}

func (r *Rooms) writer(c *Client) {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (r *Rooms) reader(c *Client, key string, rm *room) {
	defer func() {
		rm.mu.Lock()
		delete(rm.members, c)
		rm.mu.Unlock()
		close(c.send)
		c.conn.Close()
		// GC empty room
		rm.mu.RLock()
		empty := len(rm.members) == 0
		rm.mu.RUnlock()
		if empty {
			r.mu.Lock()
			delete(r.by, key)
			r.mu.Unlock()
		}
	}()
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		// For group rooms, fan out client messages to peers.
		var msg any
		if json.Unmarshal(raw, &msg) != nil {
			continue
		}
		rm.mu.RLock()
		for peer := range rm.members {
			if peer == c {
				continue
			}
			select {
			case peer.send <- raw:
			default:
			}
		}
		rm.mu.RUnlock()
	}
}
