// Package advisory — travel alerts with WebSocket broadcast.
package advisory

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/internal/ws"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct {
	pool *pgxpool.Pool
	hub  *ws.Hub
}

func NewService(pool *pgxpool.Pool, hub *ws.Hub) *Service {
	return &Service{pool: pool, hub: hub}
}

type Advisory struct {
	ID         string    `json:"id"`
	Severity   string    `json:"severity"`
	Category   string    `json:"category"`
	Title      string    `json:"title"`
	Body       *string   `json:"body,omitempty"`
	Source     *string   `json:"source,omitempty"`
	Affected   *string   `json:"affected,omitempty"`
	Confidence int       `json:"confidence"`
	ValidUntil time.Time `json:"valid_until"`
	CreatedAt  time.Time `json:"created_at"`
}

// GET /v1/advisories
func (s *Service) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, severity, category, title, body, source, affected,
		       confidence, effective_to, created_at
		FROM advisories
		WHERE effective_to > now()
		  AND ($1 = '' OR severity = $1)
		  AND ($2 = '' OR category = $2)
		ORDER BY CASE severity WHEN 'critical' THEN 0 WHEN 'warning' THEN 1 ELSE 2 END,
		         created_at DESC
	`, q.Get("severity"), q.Get("category"))
	if err != nil { response.Internal(w, err); return }
	defer rows.Close()
	out := []Advisory{}
	for rows.Next() {
		var a Advisory
		_ = rows.Scan(&a.ID, &a.Severity, &a.Category, &a.Title, &a.Body, &a.Source, &a.Affected, &a.Confidence, &a.ValidUntil, &a.CreatedAt)
		out = append(out, a)
	}
	response.OK(w, out)
}

// GET /v1/advisories/destination/{id}
func (s *Service) ForDestination(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, severity, category, title, body, source, affected,
		       confidence, effective_to, created_at
		FROM advisories
		WHERE effective_to > now()
		  AND ((scope = 'destination' AND scope_id::text = $1) OR scope = 'region')
		ORDER BY CASE severity WHEN 'critical' THEN 0 WHEN 'warning' THEN 1 ELSE 2 END
	`, id)
	if err != nil { response.Internal(w, err); return }
	defer rows.Close()
	out := []Advisory{}
	for rows.Next() {
		var a Advisory
		_ = rows.Scan(&a.ID, &a.Severity, &a.Category, &a.Title, &a.Body, &a.Source, &a.Affected, &a.Confidence, &a.ValidUntil, &a.CreatedAt)
		out = append(out, a)
	}
	response.OK(w, out)
}

// GET /v1/roads/status
func (s *Service) RoadStatus(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, name, slug, current_status, closure_reason, last_checked
		FROM roads ORDER BY name
	`)
	if err != nil { response.Internal(w, err); return }
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, name, slug, status string
		var reason *string
		var checked time.Time
		_ = rows.Scan(&id, &name, &slug, &status, &reason, &checked)
		out = append(out, map[string]any{
			"id": id, "name": name, "slug": slug, "status": status,
			"closure_reason": reason, "last_checked": checked,
		})
	}
	response.OK(w, out)
}

/* ─── Admin ──────────────────────────────────────────────── */

type adminAdvisory struct {
	Severity   string  `json:"severity"`
	Category   string  `json:"category"`
	Title      string  `json:"title"`
	Body       *string `json:"body"`
	Source     *string `json:"source"`
	Affected   *string `json:"affected"`
	Confidence *int    `json:"confidence"`
	ValidHours int     `json:"valid_hours"`
}

// POST /v1/admin/advisories — creates + broadcasts via WebSocket.
func (s *Service) AdminCreate(w http.ResponseWriter, r *http.Request) {
	var body adminAdvisory
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid body"); return
	}
	if body.ValidHours <= 0 { body.ValidHours = 48 }
	conf := 100
	if body.Confidence != nil { conf = *body.Confidence }

	var a Advisory
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO advisories (severity, category, title, body, source, affected,
		                        confidence, effective_from, effective_to)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now() + ($8 || ' hours')::interval)
		RETURNING id::text, severity, category, title, body, source, affected,
		          confidence, effective_to, created_at
	`, body.Severity, body.Category, body.Title, body.Body, body.Source, body.Affected,
		conf, body.ValidHours,
	).Scan(&a.ID, &a.Severity, &a.Category, &a.Title, &a.Body, &a.Source, &a.Affected, &a.Confidence, &a.ValidUntil, &a.CreatedAt)
	if err != nil { response.Internal(w, err); return }

	// Live-push to every connected mobile client.
	s.hub.Broadcast(map[string]any{
		"type":     "advisory.new",
		"advisory": a,
	})

	response.Created(w, a)
}

func (s *Service) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body adminAdvisory
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid body"); return
	}
	if _, err := s.pool.Exec(r.Context(), `
		UPDATE advisories SET severity=$2, category=$3, title=$4, body=$5,
		                      source=$6, affected=$7
		WHERE id=$1
	`, id, body.Severity, body.Category, body.Title, body.Body, body.Source, body.Affected); err != nil {
		response.Internal(w, err); return
	}
	s.hub.Broadcast(map[string]any{"type": "advisory.update", "id": id})
	response.OK(w, map[string]bool{"updated": true})
}

func (s *Service) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := s.pool.Exec(r.Context(),
		`UPDATE advisories SET effective_to = now() WHERE id=$1`, id); err != nil {
		response.Internal(w, err); return
	}
	s.hub.Broadcast(map[string]any{"type": "advisory.clear", "id": id})
	response.NoContent(w)
}

func (s *Service) AdminUpdateRoad(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Status        string  `json:"status"`
		ClosureReason *string `json:"closure_reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid body"); return
	}
	if _, err := s.pool.Exec(r.Context(), `
		UPDATE roads SET current_status=$2, closure_reason=$3, last_checked=now()
		WHERE id=$1
	`, id, body.Status, body.ClosureReason); err != nil {
		response.Internal(w, err); return
	}
	s.hub.Broadcast(map[string]any{"type": "road.status", "id": id, "status": body.Status})
	response.OK(w, map[string]string{"status": body.Status})
}
