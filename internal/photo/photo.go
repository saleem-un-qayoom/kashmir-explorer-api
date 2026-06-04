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

// PhotoSpot / PhotoSpotInput are OpenAPI/codegen models; handlers emit/accept
// these fields (the implementation uses inline maps/structs).
type PhotoSpot struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	DestinationSlug   string  `json:"destination_slug,omitempty"`
	Lat               float64 `json:"lat"`
	Lng               float64 `json:"lng"`
	BestTime          string  `json:"best_time"`
	Facing            string  `json:"facing"`
	TripodRecommended bool    `json:"tripod_recommended"`
	DroneAllowed      bool    `json:"drone_allowed"`
	Description       string  `json:"description"`
}

type PhotoSpotInput struct {
	DestinationSlug   string  `json:"destination_slug"`
	Name              string  `json:"name"`
	Lat               float64 `json:"lat"`
	Lng               float64 `json:"lng"`
	BestTime          string  `json:"best_time"`
	Facing            string  `json:"facing"`
	TripodRecommended bool    `json:"tripod_recommended"`
	DroneAllowed      bool    `json:"drone_allowed"`
	Description       string  `json:"description"`
}

// ForDestination godoc
// @Summary  Photo spots near a destination
// @Tags     photo-spots
// @Produce  json
// @Param    slug path string true "Destination slug"
// @Success  200 {object} response.Envelope{data=[]photo.PhotoSpot}
// @Router   /v1/destinations/{slug}/photo-spots [get]
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
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id, name, best, facing, desc string
		var lng, lat float64
		var tripod, drone bool
		if err := rows.Scan(&id, &name, &lng, &lat, &best, &facing, &tripod, &drone, &desc); err != nil {
			response.Internal(w, err)
			return
		}
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

// AdminList godoc
// @Summary  List all photo spots (admin)
// @Tags     admin-photo-spots
// @Security BearerAuth
// @Produce  json
// @Success  200 {object} response.Envelope{data=[]photo.PhotoSpot}
// @Router   /v1/admin/photo-spots [get]
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
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()

	type spot struct {
		ID              string  `json:"id"`
		Name            string  `json:"name"`
		DestinationSlug string  `json:"destination_slug"`
		Lat             float64 `json:"lat"`
		Lng             float64 `json:"lng"`
		BestTime        string  `json:"best_time"`
		Facing          string  `json:"facing"`
		TripodRec       bool    `json:"tripod_recommended"`
		DroneAllowed    bool    `json:"drone_allowed"`
		Description     string  `json:"description"`
	}
	out := []spot{}
	for rows.Next() {
		var s spot
		if err := rows.Scan(&s.ID, &s.Name, &s.DestinationSlug, &s.Lng, &s.Lat,
			&s.BestTime, &s.Facing, &s.TripodRec, &s.DroneAllowed, &s.Description); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, s)
	}
	response.OK(w, out)
}

// AdminGet godoc
// @Summary  Get a photo spot (admin)
// @Tags     admin-photo-spots
// @Security BearerAuth
// @Produce  json
// @Param    id path string true "Photo spot ID"
// @Success  200 {object} response.Envelope{data=photo.PhotoSpot}
// @Router   /v1/admin/photo-spots/{id} [get]
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
		response.Internal(w, err)
		return
	}
	response.OK(w, p)
}

// AdminCreate godoc
// @Summary  Create a photo spot (admin)
// @Tags     admin-photo-spots
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body photo.PhotoSpotInput true "Photo spot"
// @Success  201 {object} response.Envelope
// @Failure  400 {object} response.Envelope
// @Router   /v1/admin/photo-spots [post]
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
		response.BadRequest(w, "invalid body")
		return
	}
	if in.Name == "" || in.DestinationSlug == "" {
		response.BadRequest(w, "name and destination_slug required")
		return
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
		response.Internal(w, err)
		return
	}
	response.Created(w, map[string]string{"id": id})
}

// AdminUpdate godoc
// @Summary  Update a photo spot (admin)
// @Tags     admin-photo-spots
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    id   path string               true "Photo spot ID"
// @Param    body body photo.PhotoSpotInput true "Photo spot"
// @Success  200 {object} response.Envelope
// @Router   /v1/admin/photo-spots/{id} [put]
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
		response.BadRequest(w, "invalid body")
		return
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
		response.Internal(w, err)
		return
	}
	response.OK(w, map[string]string{"updated": id})
}

// AdminDelete godoc
// @Summary  Delete a photo spot (admin)
// @Tags     admin-photo-spots
// @Security BearerAuth
// @Param    id path string true "Photo spot ID"
// @Success  204
// @Router   /v1/admin/photo-spots/{id} [delete]
func (s *Service) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.pool.Exec(r.Context(), `DELETE FROM photo_spots WHERE id = $1`, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.NoContent(w)
}
