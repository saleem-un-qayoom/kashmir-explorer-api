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

// List godoc
// @Summary  List active advisories
// @Tags     advisories
// @Produce  json
// @Param    severity query string false "Filter by severity (critical, warning, info)"
// @Param    category query string false "Filter by category"
// @Success  200 {object} response.Envelope{data=[]advisory.Advisory}
// @Router   /v1/advisories [get]
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
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()
	out := []Advisory{}
	for rows.Next() {
		var a Advisory
		if err := rows.Scan(&a.ID, &a.Severity, &a.Category, &a.Title, &a.Body, &a.Source, &a.Affected, &a.Confidence, &a.ValidUntil, &a.CreatedAt); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, a)
	}
	response.OK(w, out)
}

// ForDestination godoc
// @Summary  List advisories affecting a destination
// @Tags     advisories
// @Produce  json
// @Param    id path string true "Destination ID"
// @Success  200 {object} response.Envelope{data=[]advisory.Advisory}
// @Router   /v1/advisories/destination/{id} [get]
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
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()
	out := []Advisory{}
	for rows.Next() {
		var a Advisory
		if err := rows.Scan(&a.ID, &a.Severity, &a.Category, &a.Title, &a.Body, &a.Source, &a.Affected, &a.Confidence, &a.ValidUntil, &a.CreatedAt); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, a)
	}
	response.OK(w, out)
}

// RoadStatus godoc
// @Summary  List road / mountain-pass statuses
// @Tags     advisories
// @Produce  json
// @Success  200 {object} response.Envelope
// @Router   /v1/roads/status [get]
func (s *Service) RoadStatus(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, name, slug, current_status, closure_reason, last_checked
		FROM roads ORDER BY name
	`)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, name, slug, status string
		var reason *string
		var checked time.Time
		if err := rows.Scan(&id, &name, &slug, &status, &reason, &checked); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, map[string]any{
			"id": id, "name": name, "slug": slug, "status": status,
			"closure_reason": reason, "last_checked": checked,
		})
	}
	response.OK(w, out)
}

/* ─── Admin ──────────────────────────────────────────────── */

type AdminAdvisoryInput struct {
	Severity   string  `json:"severity"`
	Category   string  `json:"category"`
	Title      string  `json:"title"`
	Body       *string `json:"body"`
	Source     *string `json:"source"`
	Affected   *string `json:"affected"`
	Confidence *int    `json:"confidence"`
	ValidHours int     `json:"valid_hours"`
}

// AdminCreate godoc
// @Summary  Create + broadcast an advisory (admin)
// @Tags     admin-advisories
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body advisory.AdminAdvisoryInput true "Advisory"
// @Success  201 {object} response.Envelope{data=advisory.Advisory}
// @Failure  400 {object} response.Envelope
// @Router   /v1/admin/advisories [post]
func (s *Service) AdminCreate(w http.ResponseWriter, r *http.Request) {
	var body AdminAdvisoryInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	if body.ValidHours <= 0 {
		body.ValidHours = 48
	}
	conf := 100
	if body.Confidence != nil {
		conf = *body.Confidence
	}

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
	if err != nil {
		response.Internal(w, err)
		return
	}

	// Live-push to every connected mobile client.
	s.hub.Broadcast(map[string]any{
		"type":     "advisory.new",
		"advisory": a,
	})

	response.Created(w, a)
}

// AdminUpdate godoc
// @Summary  Update an advisory (admin)
// @Tags     admin-advisories
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    id   path string                      true "Advisory ID"
// @Param    body body advisory.AdminAdvisoryInput true "Advisory"
// @Success  200 {object} response.Envelope
// @Router   /v1/admin/advisories/{id} [put]
func (s *Service) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body AdminAdvisoryInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `
		UPDATE advisories SET severity=$2, category=$3, title=$4, body=$5,
		                      source=$6, affected=$7
		WHERE id=$1
	`, id, body.Severity, body.Category, body.Title, body.Body, body.Source, body.Affected); err != nil {
		response.Internal(w, err)
		return
	}
	s.hub.Broadcast(map[string]any{"type": "advisory.update", "id": id})
	response.OK(w, map[string]bool{"updated": true})
}

// AdminDelete godoc
// @Summary  Expire (clear) an advisory (admin)
// @Tags     admin-advisories
// @Security BearerAuth
// @Param    id path string true "Advisory ID"
// @Success  204
// @Router   /v1/admin/advisories/{id} [delete]
func (s *Service) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := s.pool.Exec(r.Context(),
		`UPDATE advisories SET effective_to = now() WHERE id=$1`, id); err != nil {
		response.Internal(w, err)
		return
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
		response.BadRequest(w, "invalid body")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `
		UPDATE roads SET current_status=$2, closure_reason=$3, last_checked=now()
		WHERE id=$1
	`, id, body.Status, body.ClosureReason); err != nil {
		response.Internal(w, err)
		return
	}
	s.hub.Broadcast(map[string]any{"type": "road.status", "id": id, "status": body.Status})
	response.OK(w, map[string]string{"status": body.Status})
}

// ─── Admin: Roads CRUD (route-aligned) ─────────────────────────

func (s *Service) AdminRoadGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var road struct {
		ID            string    `json:"id"`
		Name          string    `json:"name"`
		Slug          string    `json:"slug"`
		Status        string    `json:"status"`
		ClosureReason *string   `json:"closure_reason"`
		LastChecked   time.Time `json:"last_checked"`
	}
	err := s.pool.QueryRow(r.Context(), `
		SELECT id::text, name, slug, current_status, closure_reason, last_checked
		FROM roads WHERE id = $1
	`, id).Scan(&road.ID, &road.Name, &road.Slug, &road.Status, &road.ClosureReason, &road.LastChecked)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, road)
}

func (s *Service) AdminRoadCreate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name          string  `json:"name"`
		Slug          string  `json:"slug"`
		Status        string  `json:"status"`
		ClosureReason *string `json:"closure_reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	if in.Name == "" || in.Slug == "" {
		response.BadRequest(w, "name and slug required")
		return
	}
	var id string
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO roads (name, slug, current_status, closure_reason)
		VALUES ($1, $2, $3, $4) RETURNING id::text
	`, in.Name, in.Slug, in.Status, in.ClosureReason).Scan(&id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.Created(w, map[string]string{"id": id})
}

func (s *Service) AdminRoadUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in struct {
		Name          string  `json:"name"`
		Slug          string  `json:"slug"`
		Status        string  `json:"status"`
		ClosureReason *string `json:"closure_reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	_, err := s.pool.Exec(r.Context(), `
		UPDATE roads SET name=$2, slug=$3, current_status=$4, closure_reason=$5, last_checked=now()
		WHERE id=$1
	`, id, in.Name, in.Slug, in.Status, in.ClosureReason)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, map[string]string{"updated": id})
}

func (s *Service) AdminRoadDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.pool.Exec(r.Context(), `DELETE FROM roads WHERE id = $1`, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.NoContent(w)
}
