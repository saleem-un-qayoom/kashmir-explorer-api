// Package image — admin CRUD for destination/trek images.
package image

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

type Image struct {
	ID            string    `json:"id"`
	DestinationID *string   `json:"destination_id"`
	TrekID        *string   `json:"trek_id"`
	URL           string    `json:"url"`
	Blurhash      *string   `json:"blurhash,omitempty"`
	Caption       *string   `json:"caption,omitempty"`
	IsHero        bool      `json:"is_hero"`
	SortOrder     int       `json:"sort_order"`
	CreatedAt     time.Time `json:"created_at"`
}

// ImageInput is the admin create/update body (OpenAPI/codegen model).
type ImageInput struct {
	DestinationID *string `json:"destination_id"`
	TrekID        *string `json:"trek_id"`
	URL           string  `json:"url"`
	Blurhash      *string `json:"blurhash"`
	Caption       *string `json:"caption"`
	IsHero        bool    `json:"is_hero"`
	SortOrder     int     `json:"sort_order"`
}

// ForDestination godoc
// @Summary  List images for a destination
// @Tags     images
// @Produce  json
// @Param    id path string true "Destination ID"
// @Success  200 {object} response.Envelope{data=[]image.Image}
// @Router   /v1/images/destination/{id} [get]
func (s *Service) ForDestination(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, destination_id::text, trek_id::text, url, blurhash,
		       caption, is_hero, sort_order, created_at
		FROM images WHERE destination_id = $1 ORDER BY sort_order, created_at
	`, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()

	out := []Image{}
	for rows.Next() {
		var img Image
		if err := rows.Scan(&img.ID, &img.DestinationID, &img.TrekID, &img.URL, &img.Blurhash,
			&img.Caption, &img.IsHero, &img.SortOrder, &img.CreatedAt); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, img)
	}
	response.OK(w, out)
}

// ForTrek godoc
// @Summary  List images for a trek
// @Tags     images
// @Produce  json
// @Param    id path string true "Trek ID"
// @Success  200 {object} response.Envelope{data=[]image.Image}
// @Router   /v1/images/trek/{id} [get]
func (s *Service) ForTrek(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, destination_id::text, trek_id::text, url, blurhash,
		       caption, is_hero, sort_order, created_at
		FROM images WHERE trek_id = $1 ORDER BY sort_order, created_at
	`, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()

	out := []Image{}
	for rows.Next() {
		var img Image
		if err := rows.Scan(&img.ID, &img.DestinationID, &img.TrekID, &img.URL, &img.Blurhash,
			&img.Caption, &img.IsHero, &img.SortOrder, &img.CreatedAt); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, img)
	}
	response.OK(w, out)
}

// AdminCreate godoc
// @Summary  Create an image record (admin)
// @Tags     admin-images
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body image.ImageInput true "Image"
// @Success  201 {object} response.Envelope
// @Failure  400 {object} response.Envelope
// @Router   /v1/admin/images [post]
func (s *Service) AdminCreate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		DestinationID *string `json:"destination_id"`
		TrekID        *string `json:"trek_id"`
		URL           string  `json:"url"`
		Blurhash      *string `json:"blurhash"`
		Caption       *string `json:"caption"`
		IsHero        bool    `json:"is_hero"`
		SortOrder     int     `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	if in.URL == "" {
		response.BadRequest(w, "url required")
		return
	}
	var id string
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO images (destination_id, trek_id, url, blurhash, caption, is_hero, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id::text
	`, in.DestinationID, in.TrekID, in.URL, in.Blurhash, in.Caption, in.IsHero, in.SortOrder).Scan(&id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.Created(w, map[string]string{"id": id})
}

// AdminUpdate godoc
// @Summary  Update image metadata (caption/hero/order) (admin)
// @Tags     admin-images
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    id   path string          true "Image ID"
// @Param    body body image.ImageInput true "Image"
// @Success  200 {object} response.Envelope
// @Router   /v1/admin/images/{id} [put]
func (s *Service) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in struct {
		Caption   *string `json:"caption"`
		IsHero    bool    `json:"is_hero"`
		SortOrder int     `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	_, err := s.pool.Exec(r.Context(), `
		UPDATE images SET caption=$2, is_hero=$3, sort_order=$4 WHERE id=$1
	`, id, in.Caption, in.IsHero, in.SortOrder)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, map[string]string{"updated": id})
}

// AdminDelete godoc
// @Summary  Delete an image (admin)
// @Tags     admin-images
// @Security BearerAuth
// @Param    id path string true "Image ID"
// @Success  204
// @Router   /v1/admin/images/{id} [delete]
func (s *Service) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.pool.Exec(r.Context(), `DELETE FROM images WHERE id = $1`, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.NoContent(w)
}
