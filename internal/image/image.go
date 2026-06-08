// Package image — admin CRUD for destination/trek images.
package image

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/pkg/response"
)

// maxUploadBytes caps an in-DB image blob. Storing bytes in Postgres is an
// interim measure ("for now") until object storage is wired up, so we keep
// the ceiling conservative.
const maxUploadBytes = 8 << 20 // 8 MiB

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

type Image struct {
	ID            string    `json:"id"`
	DestinationID *string   `json:"destination_id"`
	TrekID        *string   `json:"trek_id"`
	PhotoSpotID   *string   `json:"photo_spot_id"`
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
	PhotoSpotID   *string `json:"photo_spot_id"`
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
		SELECT id::text, destination_id::text, trek_id::text, photo_spot_id::text, url, blurhash,
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
		if err := rows.Scan(&img.ID, &img.DestinationID, &img.TrekID, &img.PhotoSpotID, &img.URL, &img.Blurhash,
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
		SELECT id::text, destination_id::text, trek_id::text, photo_spot_id::text, url, blurhash,
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
		if err := rows.Scan(&img.ID, &img.DestinationID, &img.TrekID, &img.PhotoSpotID, &img.URL, &img.Blurhash,
			&img.Caption, &img.IsHero, &img.SortOrder, &img.CreatedAt); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, img)
	}
	response.OK(w, out)
}

// ForPhotoSpot godoc
// @Summary  List images for a photo spot
// @Tags     images
// @Produce  json
// @Param    id path string true "Photo Spot ID"
// @Success  200 {object} response.Envelope{data=[]image.Image}
// @Router   /v1/images/photo-spot/{id} [get]
func (s *Service) ForPhotoSpot(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, destination_id::text, trek_id::text, photo_spot_id::text, url, blurhash,
		       caption, is_hero, sort_order, created_at
		FROM images WHERE photo_spot_id = $1 ORDER BY sort_order, created_at
	`, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()

	out := []Image{}
	for rows.Next() {
		var img Image
		if err := rows.Scan(&img.ID, &img.DestinationID, &img.TrekID, &img.PhotoSpotID, &img.URL, &img.Blurhash,
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
		PhotoSpotID   *string `json:"photo_spot_id"`
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
		INSERT INTO images (destination_id, trek_id, photo_spot_id, url, blurhash, caption, is_hero, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id::text
	`, in.DestinationID, in.TrekID, in.PhotoSpotID, in.URL, in.Blurhash, in.Caption, in.IsHero, in.SortOrder).Scan(&id)
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

// UploadResult is the response body for a direct binary upload.
type UploadResult struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// Upload godoc
// @Summary  Upload an image, stored directly in the DB (interim)
// @Description Accepts a multipart form ("file") or a raw image body. The bytes
// @Description are stored in Postgres and served back via /v1/images/{id}/raw.
// @Tags     upload
// @Security BearerAuth
// @Accept   multipart/form-data
// @Produce  json
// @Param    file           formData file   true  "Image file"
// @Param    destination_id formData string false "Destination ID"
// @Param    trek_id        formData string false "Trek ID"
// @Param    caption        formData string false "Caption"
// @Param    is_hero        formData bool   false "Is hero image"
// @Param    sort_order     formData int    false "Sort order"
// @Success  201 {object} response.Envelope{data=image.UploadResult}
// @Failure  400 {object} response.Envelope
// @Router   /v1/upload/image [post]
func (s *Service) Upload(w http.ResponseWriter, r *http.Request) {
	var (
		data        []byte
		contentType string
		destID      *string
		trekID      *string
		photoSpotID *string
		caption     *string
		isHero      bool
		sortOrder   int
		err         error
	)

	if file, hdr, ferr := r.FormFile("file"); ferr == nil {
		// Multipart form upload.
		defer file.Close()
		data, err = io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
		if err != nil {
			response.Internal(w, err)
			return
		}
		contentType = hdr.Header.Get("Content-Type")
		destID = optStr(r.FormValue("destination_id"))
		trekID = optStr(r.FormValue("trek_id"))
		photoSpotID = optStr(r.FormValue("photo_spot_id"))
		caption = optStr(r.FormValue("caption"))
		isHero, _ = strconv.ParseBool(r.FormValue("is_hero"))
		sortOrder, _ = strconv.Atoi(r.FormValue("sort_order"))
	} else {
		// Raw image body; metadata via query params.
		data, err = io.ReadAll(io.LimitReader(r.Body, maxUploadBytes+1))
		if err != nil {
			response.Internal(w, err)
			return
		}
		q := r.URL.Query()
		contentType = r.Header.Get("Content-Type")
		destID = optStr(q.Get("destination_id"))
		trekID = optStr(q.Get("trek_id"))
		photoSpotID = optStr(q.Get("photo_spot_id"))
		caption = optStr(q.Get("caption"))
		isHero, _ = strconv.ParseBool(q.Get("is_hero"))
		sortOrder, _ = strconv.Atoi(q.Get("sort_order"))
	}

	if len(data) == 0 {
		response.BadRequest(w, "empty file")
		return
	}
	if len(data) > maxUploadBytes {
		response.BadRequest(w, "file too large (max 8 MiB)")
		return
	}
	if !allowedType(contentType) {
		response.BadRequest(w, "content type not allowed")
		return
	}

	// Insert the blob, then backfill url with the self-referential serve path
	// so existing list endpoints (which read `url`) keep working unchanged.
	var (
		id  string
		url string
	)
	// Generate the id up front so we can store the self-referential serve URL
	// in the same INSERT. (A data-modifying CTE's inserted row isn't visible to
	// a sibling UPDATE in the same statement, so we can't backfill url that way.)
	err = s.pool.QueryRow(r.Context(), `
		WITH new AS (SELECT uuid_generate_v4() AS id)
		INSERT INTO images (id, destination_id, trek_id, photo_spot_id, url, data, content_type,
		                    caption, is_hero, sort_order)
		SELECT new.id, $1, $2, $3, '/v1/images/' || new.id::text || '/raw',
		       $4, $5, $6, $7, $8
		FROM new
		RETURNING id::text, url
	`, destID, trekID, photoSpotID, data, contentType, caption, isHero, sortOrder).Scan(&id, &url)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.Created(w, UploadResult{ID: id, URL: url})
}

// Raw godoc
// @Summary  Serve raw image bytes stored in the DB
// @Tags     images
// @Produce  image/*
// @Param    id path string true "Image ID"
// @Success  200 {file} binary
// @Failure  404 {object} response.Envelope
// @Router   /v1/images/{id}/raw [get]
func (s *Service) Raw(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var (
		data        []byte
		contentType *string
	)
	err := s.pool.QueryRow(r.Context(), `
		SELECT data, content_type FROM images WHERE id = $1
	`, id).Scan(&data, &contentType)
	if err != nil {
		response.NotFound(w, "image not found")
		return
	}
	if len(data) == 0 {
		response.NotFound(w, "image has no stored bytes")
		return
	}
	ct := "application/octet-stream"
	if contentType != nil && *contentType != "" {
		ct = *contentType
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func optStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func allowedType(ct string) bool {
	switch ct {
	case "image/jpeg", "image/png", "image/webp", "image/avif", "image/heic":
		return true
	}
	return false
}
