// Package sync — apply queued offline mutations from the mobile client.
//
// Mobile queues `save`/`unsave`/`add_to_trip` operations while offline; on
// reconnect it POSTs the queue here. We replay each op against the live
// state, idempotently.
package sync

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	mw "github.com/kashmir-explorer/api/internal/middleware"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

type op struct {
	Op      string          `json:"op"`
	Payload json.RawMessage `json:"payload"`
	Ts      int64           `json:"ts"`
}
type req struct {
	Ops []op `json:"ops"`
}

// POST /v1/sync
func (s *Service) Apply(w http.ResponseWriter, r *http.Request) {
	userID := mw.UserID(r)
	var body req
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}

	applied := 0
	for _, o := range body.Ops {
		switch o.Op {
		case "save":
			var p struct {
				DestinationID string `json:"destination_id"`
			}
			if json.Unmarshal(o.Payload, &p) != nil || p.DestinationID == "" {
				continue
			}
			_, err := s.pool.Exec(r.Context(),
				`INSERT INTO saved_destinations (user_id, destination_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				userID, p.DestinationID)
			if err == nil {
				applied++
			}

		case "unsave":
			var p struct {
				DestinationID string `json:"destination_id"`
			}
			if json.Unmarshal(o.Payload, &p) != nil || p.DestinationID == "" {
				continue
			}
			_, err := s.pool.Exec(r.Context(),
				`DELETE FROM saved_destinations WHERE user_id=$1 AND destination_id=$2`,
				userID, p.DestinationID)
			if err == nil {
				applied++
			}
		}

		// Also enqueue audit row so admins can see what was replayed.
		_, _ = s.pool.Exec(r.Context(),
			`INSERT INTO sync_queue (user_id, op, payload, applied_at) VALUES ($1, $2, $3, now())`,
			userID, o.Op, o.Payload)
	}

	response.OK(w, map[string]any{"applied": applied})
}
