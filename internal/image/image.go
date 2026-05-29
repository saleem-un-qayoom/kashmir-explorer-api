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

// GET /v1/images/destination/{id}
func (s *Service) ForDestination(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, destination_id::text, trek_id::text, url, blurhash,
		       caption, is_hero, sort_order, created_at
		FROM images WHERE destination_id = $1 ORDER BY sort_order, created_at
	`, id)
	if err != nil { response.Internal(w, err); return }
	defer rows.Close()

	out := []Image{}
	for rows.Next() {
		var img Image
		_ = rows.Scan(&img.ID, &img.DestinationID, &img.TrekID, &img.URL, &img.Blurhash,
			&img.Caption, &img.IsHero, &img.SortOrder, &img.CreatedAt)
		out = append(out, img)
	}
	response.OK(w, out)
}

// GET /v1/images/trek/{id}
func (s *Service) ForTrek(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, destination_id::text, trek_id::text, url, blurhash,
		       caption, is_hero, sort_order, created_at
		FROM images WHERE trek_id = $1 ORDER BY sort_order, created_at
	`, id)
	if err != nil { response.Internal(w, err); return }
	defer rows.Close()

	out := []Image{}
	for rows.Next() {
		var img Image
		_ = rows.Scan(&img.ID, &img.DestinationID, &img.TrekID, &img.URL, &img.Blurhash,
			&img.Caption, &img.IsHero, &img.SortOrder, &img.CreatedAt)
		out = append(out, img)
	}
	response.OK(w, out)
}

// POST /v1/admin/images — create image record.
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
		response.BadRequest(w, "invalid body"); return
	}
	if in.URL == "" {
		response.BadRequest(w, "url required"); return
	}
	var id string
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO images (destination_id, trek_id, url, blurhash, caption, is_hero, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id::text
	`, in.DestinationID, in.TrekID, in.URL, in.Blurhash, in.Caption, in.IsHero, in.SortOrder).Scan(&id)
	if err != nil {
		response.Internal(w, err); return
	}
	response.Created(w, map[string]string{"id": id})
}

// PUT /v1/admin/images/{id}
func (s *Service) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in struct {
		Caption   *string `json:"caption"`
		IsHero    bool    `json:"is_hero"`
		SortOrder int     `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body"); return
	}
	_, err := s.pool.Exec(r.Context(), `
		UPDATE images SET caption=$2, is_hero=$3, sort_order=$4 WHERE id=$1
	`, id, in.Caption, in.IsHero, in.SortOrder)
	if err != nil {
		response.Internal(w, err); return
	}
	response.OK(w, map[string]string{"updated": id})
}

// DELETE /v1/admin/images/{id}
func (s *Service) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.pool.Exec(r.Context(), `DELETE FROM images WHERE id = $1`, id)
	if err != nil {
		response.Internal(w, err); return
	}
	response.NoContent(w)
}
