// Package crowd — aggregate trek-nav pings into an anonymised density signal.
//
// Mobile pings POST /v1/treks/{slug}/ping every ~60s while navigating with
// only along-track distance + the slug. We aggregate over the last 4 hours
// to surface: how many people are currently on this trek, and how many are
// near the user's segment.
package crowd

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	mw "github.com/kashmir-explorer/api/internal/middleware"
	"github.com/kashmir-explorer/api/internal/ws"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct {
	pool  *pgxpool.Pool
	rooms *ws.Rooms
}

func NewService(pool *pgxpool.Pool, rooms *ws.Rooms) *Service {
	return &Service{pool: pool, rooms: rooms}
}

type pingReq struct {
	AlongM int `json:"along_m"`
}

// POST /v1/treks/{slug}/ping — auth required. Append-only.
func (s *Service) Ping(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	userID := mw.UserID(r)
	var body pingReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	if _, err := s.pool.Exec(r.Context(),
		`INSERT INTO trek_nav_pings (trek_slug, user_id, along_m) VALUES ($1, $2, $3)`,
		slug, userID, body.AlongM); err != nil {
		response.Internal(w, err)
		return
	}

	// Recompute + broadcast density to this trek's crowd room.
	d, _ := s.computeDensity(r.Context(), slug)
	s.rooms.BroadcastRoom("crowd:"+slug, map[string]any{
		"type":    "density",
		"trek":    slug,
		"density": d,
	})
	response.NoContent(w)
}

// GET /v1/treks/{slug}/density — public anonymised snapshot.
func (s *Service) Density(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	d, err := s.computeDensity(r.Context(), slug)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, d)
}

type Density struct {
	ActiveTrekkers int       `json:"active_trekkers"`
	AheadOfUser    *int      `json:"ahead_of_user,omitempty"` // populated when ?along_m=
	UpdatedAt      time.Time `json:"updated_at"`
}

func (s *Service) computeDensity(ctx context.Context, slug string) (Density, error) {
	d := Density{UpdatedAt: time.Now().UTC()}
	if err := s.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT user_id) FROM trek_nav_pings
		WHERE trek_slug = $1
		  AND recorded_at > now() - INTERVAL '4 hours'
	`, slug).Scan(&d.ActiveTrekkers); err != nil {
		return d, err
	}
	return d, nil
}

/* ─── Cleanup job (called from jobs package) ─────────────── */

// PurgeOldPings — call hourly to keep the table bounded.
func PurgeOldPings(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	tag, err := pool.Exec(ctx,
		`DELETE FROM trek_nav_pings WHERE recorded_at < now() - INTERVAL '6 hours'`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
