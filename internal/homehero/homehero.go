// Package homehero — curated home-screen hero banners (admin-managed carousel).
//
// Independent of destinations/treks: the mobile home screen reads the active
// banners ordered by sort_order, instead of borrowing the featured
// destination's hero image.
package homehero

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

type Banner struct {
	ID        string    `json:"id"`
	ImageURL  string    `json:"image_url"`
	Blurhash  *string   `json:"blurhash,omitempty"`
	Title     *string   `json:"title,omitempty"`
	Subtitle  *string   `json:"subtitle,omitempty"`
	LinkType  string    `json:"link_type"`  // none | destination | trek | screen
	LinkValue *string   `json:"link_value,omitempty"`
	SortOrder int       `json:"sort_order"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BannerInput is the admin create/update body (OpenAPI/codegen model).
type BannerInput struct {
	ImageURL  string  `json:"image_url"`
	Blurhash  *string `json:"blurhash"`
	Title     *string `json:"title"`
	Subtitle  *string `json:"subtitle"`
	LinkType  string  `json:"link_type"`
	LinkValue *string `json:"link_value"`
	SortOrder int     `json:"sort_order"`
	IsActive  bool    `json:"is_active"`
}

func scan(rows interface {
	Scan(dest ...any) error
}, b *Banner) error {
	return rows.Scan(&b.ID, &b.ImageURL, &b.Blurhash, &b.Title, &b.Subtitle,
		&b.LinkType, &b.LinkValue, &b.SortOrder, &b.IsActive, &b.CreatedAt, &b.UpdatedAt)
}

const cols = `id::text, image_url, blurhash, title, subtitle,
	link_type, link_value, sort_order, is_active, created_at, updated_at`

// List godoc
// @Summary  List active home hero banners
// @Tags     home-hero
// @Produce  json
// @Success  200 {object} response.Envelope{data=[]homehero.Banner}
// @Router   /v1/home-hero [get]
func (s *Service) List(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT `+cols+`
		FROM home_hero_banners WHERE is_active = true
		ORDER BY sort_order, created_at
	`)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()
	out := []Banner{}
	for rows.Next() {
		var b Banner
		if err := scan(rows, &b); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, b)
	}
	response.OK(w, out)
}

/* ─── Admin ──────────────────────────────────────────────── */

// AdminList godoc
// @Summary  List all home hero banners incl. inactive (admin)
// @Tags     admin-home-hero
// @Security BearerAuth
// @Produce  json
// @Success  200 {object} response.Envelope{data=[]homehero.Banner}
// @Router   /v1/admin/home-hero [get]
func (s *Service) AdminList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT `+cols+`
		FROM home_hero_banners ORDER BY sort_order, created_at
	`)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()
	out := []Banner{}
	for rows.Next() {
		var b Banner
		if err := scan(rows, &b); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, b)
	}
	response.OK(w, out)
}

// AdminGet godoc
// @Summary  Get a home hero banner (admin)
// @Tags     admin-home-hero
// @Security BearerAuth
// @Produce  json
// @Param    id path string true "Banner ID"
// @Success  200 {object} response.Envelope{data=homehero.Banner}
// @Router   /v1/admin/home-hero/{id} [get]
func (s *Service) AdminGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var b Banner
	err := scan(s.pool.QueryRow(r.Context(),
		`SELECT `+cols+` FROM home_hero_banners WHERE id = $1`, id), &b)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, b)
}

func normalize(in *BannerInput) {
	if in.LinkType == "" {
		in.LinkType = "none"
	}
}

// AdminCreate godoc
// @Summary  Create a home hero banner (admin)
// @Tags     admin-home-hero
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body homehero.BannerInput true "Banner"
// @Success  201 {object} response.Envelope
// @Failure  400 {object} response.Envelope
// @Router   /v1/admin/home-hero [post]
func (s *Service) AdminCreate(w http.ResponseWriter, r *http.Request) {
	var in BannerInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	if in.ImageURL == "" {
		response.BadRequest(w, "image_url required")
		return
	}
	normalize(&in)
	var id string
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO home_hero_banners
			(image_url, blurhash, title, subtitle, link_type, link_value, sort_order, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id::text
	`, in.ImageURL, in.Blurhash, in.Title, in.Subtitle, in.LinkType, in.LinkValue,
		in.SortOrder, in.IsActive).Scan(&id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.Created(w, map[string]string{"id": id})
}

// AdminUpdate godoc
// @Summary  Update a home hero banner (admin)
// @Tags     admin-home-hero
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    id   path string               true "Banner ID"
// @Param    body body homehero.BannerInput true "Banner"
// @Success  200 {object} response.Envelope
// @Router   /v1/admin/home-hero/{id} [put]
func (s *Service) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in BannerInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	normalize(&in)
	_, err := s.pool.Exec(r.Context(), `
		UPDATE home_hero_banners SET
			image_url=$2, blurhash=$3, title=$4, subtitle=$5,
			link_type=$6, link_value=$7, sort_order=$8, is_active=$9, updated_at=now()
		WHERE id=$1
	`, id, in.ImageURL, in.Blurhash, in.Title, in.Subtitle,
		in.LinkType, in.LinkValue, in.SortOrder, in.IsActive)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, map[string]string{"updated": id})
}

// AdminDelete godoc
// @Summary  Delete a home hero banner (admin)
// @Tags     admin-home-hero
// @Security BearerAuth
// @Param    id path string true "Banner ID"
// @Success  204
// @Router   /v1/admin/home-hero/{id} [delete]
func (s *Service) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := s.pool.Exec(r.Context(),
		`DELETE FROM home_hero_banners WHERE id = $1`, id); err != nil {
		response.Internal(w, err)
		return
	}
	response.NoContent(w)
}
