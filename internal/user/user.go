// Package user — /me + saved + itineraries (CRUD).
package user

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	mw "github.com/kashmir-explorer/api/internal/middleware"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

// GET /v1/me
func (s *Service) Me(w http.ResponseWriter, r *http.Request) {
	uid := mw.UserID(r)
	var name, email, phone, role *string
	err := s.pool.QueryRow(r.Context(),
		`SELECT name, email, phone, role FROM users WHERE id = $1`, uid,
	).Scan(&name, &email, &phone, &role)
	if err != nil {
		response.NotFound(w, "user not found")
		return
	}
	response.OK(w, map[string]any{"id": uid, "name": name, "email": email, "phone": phone, "role": role})
}

// PATCH /v1/me
func (s *Service) Update(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]string{"todo": "update name, language, medical, insurance"})
}

// GET /v1/saved
func (s *Service) ListSaved(w http.ResponseWriter, r *http.Request) {
	uid := mw.UserID(r)
	rows, err := s.pool.Query(r.Context(), `
		SELECT d.id::text, d.slug, d.name, d.district, d.altitude_m, d.rating, s.saved_at
		FROM saved_destinations s JOIN destinations d ON d.id = s.destination_id
		WHERE s.user_id = $1 ORDER BY s.saved_at DESC
	`, uid)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, slug, name string
		var district *string
		var alt *int
		var rating float64
		var savedAt any
		_ = rows.Scan(&id, &slug, &name, &district, &alt, &rating, &savedAt)
		out = append(out, map[string]any{"id": id, "slug": slug, "name": name, "district": district, "altitude_m": alt, "rating": rating, "saved_at": savedAt})
	}
	response.OK(w, out)
}

// POST /v1/saved/{destinationId}
func (s *Service) Save(w http.ResponseWriter, r *http.Request) {
	uid := mw.UserID(r)
	did := chi.URLParam(r, "destinationId")
	_, err := s.pool.Exec(r.Context(),
		`INSERT INTO saved_destinations (user_id, destination_id) VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`, uid, did)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, map[string]bool{"saved": true})
}

// DELETE /v1/saved/{destinationId}
func (s *Service) Unsave(w http.ResponseWriter, r *http.Request) {
	uid := mw.UserID(r)
	did := chi.URLParam(r, "destinationId")
	_, _ = s.pool.Exec(r.Context(),
		`DELETE FROM saved_destinations WHERE user_id = $1 AND destination_id = $2`, uid, did)
	response.NoContent(w)
}

// ─── Itineraries ────────────────────────────────────────────

func (s *Service) ListItineraries(w http.ResponseWriter, r *http.Request) {
	uid := mw.UserID(r)
	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, title, duration, start_date, is_public, share_token, created_at
		FROM itineraries WHERE user_id = $1 ORDER BY created_at DESC
	`, uid)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, title string
		var dur *int
		var startDate, createdAt any
		var isPublic bool
		var shareToken *string
		_ = rows.Scan(&id, &title, &dur, &startDate, &isPublic, &shareToken, &createdAt)
		out = append(out, map[string]any{"id": id, "title": title, "duration": dur, "start_date": startDate, "is_public": isPublic, "share_token": shareToken, "created_at": createdAt})
	}
	response.OK(w, out)
}

func (s *Service) CreateItinerary(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]string{"todo": "create itinerary"})
}
func (s *Service) UpdateItinerary(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]string{"todo": "update itinerary"})
}
func (s *Service) DeleteItinerary(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]string{"todo": "delete itinerary"})
}
