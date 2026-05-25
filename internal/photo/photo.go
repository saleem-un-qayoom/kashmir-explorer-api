// Package photo — photo-spot listings per destination.
package photo

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

// GET /v1/destinations/{slug}/photo-spots
func (s *Service) ForDestination(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	rows, err := s.pool.Query(r.Context(), `
		SELECT ps.id::text, ps.name,
		       ST_X(ps.location::geometry), ST_Y(ps.location::geometry),
		       COALESCE(ps.best_time, ''),
		       COALESCE(ps.facing, ''),
		       COALESCE(ps.tripod_recommended, false),
		       COALESCE(ps.drone_allowed, false),
		       COALESCE(ps.description, '')
		FROM photo_spots ps
		JOIN destinations d ON d.id = ps.destination_id
		WHERE d.slug = $1
		ORDER BY ps.name
	`, slug)
	if err != nil { response.Internal(w, err); return }
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id, name, best, facing, desc string
		var lng, lat float64
		var tripod, drone bool
		_ = rows.Scan(&id, &name, &lng, &lat, &best, &facing, &tripod, &drone, &desc)
		out = append(out, map[string]any{
			"id": id, "name": name, "lat": lat, "lng": lng,
			"best_time": best, "facing": facing,
			"tripod_recommended": tripod, "drone_allowed": drone,
			"description": desc,
		})
	}
	response.OK(w, out)
}
