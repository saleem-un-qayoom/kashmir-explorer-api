// Package photo — photo-spot listings per destination.
package photo

import (
	"encoding/json"
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

// ─── Admin CRUD ────────────────────────────────────────────────

func (s *Service) AdminList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT ps.id::text, ps.name, d.slug,
		       ST_X(ps.location::geometry), ST_Y(ps.location::geometry),
		       COALESCE(ps.best_time, ''),
		       COALESCE(ps.facing, ''),
		       COALESCE(ps.tripod_recommended, false),
		       COALESCE(ps.drone_allowed, false),
		       COALESCE(ps.description, '')
		FROM photo_spots ps
		JOIN destinations d ON d.id = ps.destination_id
		ORDER BY ps.name
	`)
	if err != nil { response.Internal(w, err); return }
	defer rows.Close()

	type spot struct {
		ID               string  `json:"id"`
		Name             string  `json:"name"`
		DestinationSlug  string  `json:"destination_slug"`
		Lat              float64 `json:"lat"`
		Lng              float64 `json:"lng"`
		BestTime         string  `json:"best_time"`
		Facing           string  `json:"facing"`
		TripodRec        bool    `json:"tripod_recommended"`
		DroneAllowed     bool    `json:"drone_allowed"`
		Description      string  `json:"description"`
	}
	out := []spot{}
	for rows.Next() {
		var s spot
		_ = rows.Scan(&s.ID, &s.Name, &s.DestinationSlug, &s.Lng, &s.Lat,
			&s.BestTime, &s.Facing, &s.TripodRec, &s.DroneAllowed, &s.Description)
		out = append(out, s)
	}
	response.OK(w, out)
}

func (s *Service) AdminGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var p struct {
		ID              string   `json:"id"`
		DestinationID   string   `json:"destination_id"`
		DestinationSlug string   `json:"destination_slug"`
		Name            string   `json:"name"`
		Lat             *float64 `json:"lat"`
		Lng             *float64 `json:"lng"`
		BestTime        string   `json:"best_time"`
		Facing          string   `json:"facing"`
		TripodRec       bool     `json:"tripod_recommended"`
		DroneAllowed    bool     `json:"drone_allowed"`
		Description     string   `json:"description"`
	}
	err := s.pool.QueryRow(r.Context(), `
		SELECT ps.id::text, ps.destination_id::text, d.slug,
		       ps.name,
		       ST_X(ps.location::geometry), ST_Y(ps.location::geometry),
		       COALESCE(ps.best_time, ''),
		       COALESCE(ps.facing, ''),
		       COALESCE(ps.tripod_recommended, false),
		       COALESCE(ps.drone_allowed, false),
		       COALESCE(ps.description, '')
		FROM photo_spots ps
		JOIN destinations d ON d.id = ps.destination_id
		WHERE ps.id = $1
	`, id).Scan(&p.ID, &p.DestinationID, &p.DestinationSlug, &p.Name,
		&p.Lng, &p.Lat, &p.BestTime, &p.Facing, &p.TripodRec, &p.DroneAllowed, &p.Description)
	if err != nil {
		response.Internal(w, err); return
	}
	response.OK(w, p)
}

func (s *Service) AdminCreate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		DestinationSlug string  `json:"destination_slug"`
		Name            string  `json:"name"`
		Lat             float64 `json:"lat"`
		Lng             float64 `json:"lng"`
		BestTime        string  `json:"best_time"`
		Facing          string  `json:"facing"`
		TripodRec       bool    `json:"tripod_recommended"`
		DroneAllowed    bool    `json:"drone_allowed"`
		Description     string  `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body"); return
	}
	if in.Name == "" || in.DestinationSlug == "" {
		response.BadRequest(w, "name and destination_slug required"); return
	}
	var id string
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO photo_spots (destination_id, name, location, best_time, facing,
		                         tripod_recommended, drone_allowed, description)
		VALUES ((SELECT id FROM destinations WHERE slug = $1), $2,
		        ST_GeogFromText('POINT(' || $3::text || ' ' || $4::text || ')'),
		        $5, $6, $7, $8, $9)
		RETURNING id::text
	`, in.DestinationSlug, in.Name, in.Lng, in.Lat,
		in.BestTime, in.Facing, in.TripodRec, in.DroneAllowed, in.Description).Scan(&id)
	if err != nil {
		response.Internal(w, err); return
	}
	response.Created(w, map[string]string{"id": id})
}

func (s *Service) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in struct {
		DestinationSlug string  `json:"destination_slug"`
		Name            string  `json:"name"`
		Lat             float64 `json:"lat"`
		Lng             float64 `json:"lng"`
		BestTime        string  `json:"best_time"`
		Facing          string  `json:"facing"`
		TripodRec       bool    `json:"tripod_recommended"`
		DroneAllowed    bool    `json:"drone_allowed"`
		Description     string  `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body"); return
	}
	_, err := s.pool.Exec(r.Context(), `
		UPDATE photo_spots SET
			destination_id = (SELECT id FROM destinations WHERE slug = $2),
			name = $3,
			location = ST_GeogFromText('POINT(' || $4::text || ' ' || $5::text || ')'),
			best_time = $6, facing = $7,
			tripod_recommended = $8, drone_allowed = $9, description = $10
		WHERE id = $1
	`, id, in.DestinationSlug, in.Name, in.Lng, in.Lat,
		in.BestTime, in.Facing, in.TripodRec, in.DroneAllowed, in.Description)
	if err != nil {
		response.Internal(w, err); return
	}
	response.OK(w, map[string]string{"updated": id})
}

func (s *Service) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.pool.Exec(r.Context(), `DELETE FROM photo_spots WHERE id = $1`, id)
	if err != nil {
		response.Internal(w, err); return
	}
	response.NoContent(w)
}
