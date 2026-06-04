// Package review — user-generated reviews + ratings for destinations and treks.
//
// One review per user per target (upsert on re-submit). The target's
// rating + review_count aggregates are recomputed from visible reviews after
// every create/delete so the discovery surfaces stay in sync.
package review

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	mw "github.com/kashmir-explorer/api/internal/middleware"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

type Review struct {
	ID        string   `json:"id"`
	Rating    int      `json:"rating"`
	Body      *string  `json:"body,omitempty"`
	Photos    []string `json:"photos,omitempty"`
	Author    string   `json:"author"`
	CreatedAt string   `json:"created_at"`
	// Populated on the "my reviews" + admin lists.
	TargetType string `json:"target_type,omitempty"`
	TargetSlug string `json:"target_slug,omitempty"`
	Hidden     bool   `json:"hidden,omitempty"`
}

type ReviewInput struct {
	Rating int      `json:"rating"`
	Body   string   `json:"body,omitempty"`
	Photos []string `json:"photos,omitempty"`
}

// targetTable maps the (trusted, route-derived) target type to its table. Never
// pass user input here — it's used in query construction.
func targetTable(targetType string) string {
	if targetType == "trek" {
		return "treks"
	}
	return "destinations"
}

func (s *Service) targetIDFromSlug(ctx context.Context, targetType, slug string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, "SELECT id::text FROM "+targetTable(targetType)+" WHERE slug = $1", slug).Scan(&id)
	return id, err
}

// recompute refreshes the target's rating + review_count from visible reviews.
func (s *Service) recompute(ctx context.Context, targetType, targetID string) {
	tbl := targetTable(targetType)
	_, _ = s.pool.Exec(ctx, `
		UPDATE `+tbl+` SET
			rating = COALESCE((SELECT ROUND(AVG(rating)::numeric, 2)
			                   FROM reviews WHERE target_type=$1 AND target_id=$2::uuid AND hidden=false), 0),
			review_count = (SELECT COUNT(*)
			                FROM reviews WHERE target_type=$1 AND target_id=$2::uuid AND hidden=false)
		WHERE id=$2::uuid`, targetType, targetID)
}

// ─── Create (upsert) ──────────────────────────────────────────

func (s *Service) create(w http.ResponseWriter, r *http.Request, targetType string) {
	userID := mw.UserID(r)
	if userID == "" {
		response.Unauthorized(w, "login required")
		return
	}
	slug := chi.URLParam(r, "slug")
	var in ReviewInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	if in.Rating < 1 || in.Rating > 5 {
		response.BadRequest(w, "rating must be between 1 and 5")
		return
	}
	tid, err := s.targetIDFromSlug(r.Context(), targetType, slug)
	if err != nil {
		response.NotFound(w, targetType+" not found")
		return
	}

	var id string
	err = s.pool.QueryRow(r.Context(), `
		INSERT INTO reviews (user_id, target_type, target_id, rating, body, photos)
		VALUES ($1, $2, $3::uuid, $4, NULLIF($5, ''), $6)
		ON CONFLICT (user_id, target_type, target_id)
		DO UPDATE SET rating = $4, body = NULLIF($5, ''), photos = $6, hidden = false, updated_at = now()
		RETURNING id::text`,
		userID, targetType, tid, in.Rating, in.Body, in.Photos).Scan(&id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	s.recompute(r.Context(), targetType, tid)
	response.Created(w, map[string]string{"id": id})
}

// CreateForDestination godoc
// @Summary  Create/update my review for a destination
// @Tags     reviews
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    slug path string            true "Destination slug"
// @Param    body body review.ReviewInput true "Review"
// @Success  201 {object} response.Envelope
// @Failure  400 {object} response.Envelope
// @Failure  404 {object} response.Envelope
// @Router   /v1/destinations/{slug}/reviews [post]
func (s *Service) CreateForDestination(w http.ResponseWriter, r *http.Request) {
	s.create(w, r, "destination")
}

// CreateForTrek godoc
// @Summary  Create/update my review for a trek
// @Tags     reviews
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    slug path string            true "Trek slug"
// @Param    body body review.ReviewInput true "Review"
// @Success  201 {object} response.Envelope
// @Failure  400 {object} response.Envelope
// @Failure  404 {object} response.Envelope
// @Router   /v1/treks/{slug}/reviews [post]
func (s *Service) CreateForTrek(w http.ResponseWriter, r *http.Request) {
	s.create(w, r, "trek")
}

// ─── List ─────────────────────────────────────────────────────

func (s *Service) list(w http.ResponseWriter, r *http.Request, targetType string) {
	slug := chi.URLParam(r, "slug")
	tid, err := s.targetIDFromSlug(r.Context(), targetType, slug)
	if err != nil {
		response.NotFound(w, targetType+" not found")
		return
	}
	rows, err := s.pool.Query(r.Context(), `
		SELECT rv.id::text, rv.rating, rv.body, rv.photos,
		       COALESCE(u.name, 'Traveller'), rv.created_at
		FROM reviews rv JOIN users u ON u.id = rv.user_id
		WHERE rv.target_type = $1 AND rv.target_id = $2::uuid AND rv.hidden = false
		ORDER BY rv.created_at DESC
		LIMIT 100`, targetType, tid)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()

	out := make([]Review, 0)
	for rows.Next() {
		var rv Review
		var created any
		if err := rows.Scan(&rv.ID, &rv.Rating, &rv.Body, &rv.Photos, &rv.Author, &created); err != nil {
			response.Internal(w, err)
			return
		}
		rv.CreatedAt = toString(created)
		out = append(out, rv)
	}
	response.OK(w, out)
}

// ListForDestination godoc
// @Summary  List reviews for a destination
// @Tags     reviews
// @Produce  json
// @Param    slug path string true "Destination slug"
// @Success  200 {object} response.Envelope{data=[]review.Review}
// @Router   /v1/destinations/{slug}/reviews [get]
func (s *Service) ListForDestination(w http.ResponseWriter, r *http.Request) {
	s.list(w, r, "destination")
}

// ListForTrek godoc
// @Summary  List reviews for a trek
// @Tags     reviews
// @Produce  json
// @Param    slug path string true "Trek slug"
// @Success  200 {object} response.Envelope{data=[]review.Review}
// @Router   /v1/treks/{slug}/reviews [get]
func (s *Service) ListForTrek(w http.ResponseWriter, r *http.Request) {
	s.list(w, r, "trek")
}

// ─── Mine ─────────────────────────────────────────────────────

// Mine godoc
// @Summary  List my reviews
// @Tags     reviews
// @Security BearerAuth
// @Produce  json
// @Success  200 {object} response.Envelope{data=[]review.Review}
// @Router   /v1/me/reviews [get]
func (s *Service) Mine(w http.ResponseWriter, r *http.Request) {
	userID := mw.UserID(r)
	if userID == "" {
		response.Unauthorized(w, "login required")
		return
	}
	rows, err := s.pool.Query(r.Context(), `
		SELECT rv.id::text, rv.rating, rv.body, rv.photos, rv.created_at, rv.target_type,
		       COALESCE((SELECT slug FROM destinations WHERE id = rv.target_id),
		                (SELECT slug FROM treks WHERE id = rv.target_id), '')
		FROM reviews rv
		WHERE rv.user_id = $1
		ORDER BY rv.created_at DESC`, userID)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()

	out := make([]Review, 0)
	for rows.Next() {
		var rv Review
		var created any
		if err := rows.Scan(&rv.ID, &rv.Rating, &rv.Body, &rv.Photos, &created, &rv.TargetType, &rv.TargetSlug); err != nil {
			response.Internal(w, err)
			return
		}
		rv.CreatedAt = toString(created)
		out = append(out, rv)
	}
	response.OK(w, out)
}

// Delete godoc
// @Summary  Delete my review
// @Tags     reviews
// @Security BearerAuth
// @Param    id path string true "Review ID"
// @Success  204
// @Failure  404 {object} response.Envelope
// @Router   /v1/reviews/{id} [delete]
func (s *Service) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := mw.UserID(r)

	var targetType, targetID string
	err := s.pool.QueryRow(r.Context(),
		`SELECT target_type, target_id::text FROM reviews WHERE id = $1::uuid AND user_id = $2`,
		id, userID).Scan(&targetType, &targetID)
	if err != nil {
		response.NotFound(w, "review not found")
		return
	}
	if _, err := s.pool.Exec(r.Context(),
		`DELETE FROM reviews WHERE id = $1::uuid AND user_id = $2`, id, userID); err != nil {
		response.Internal(w, err)
		return
	}
	s.recompute(r.Context(), targetType, targetID)
	response.NoContent(w)
}

// ─── Admin ────────────────────────────────────────────────────

// AdminList godoc
// @Summary  List all reviews (admin moderation)
// @Tags     admin-reviews
// @Security BearerAuth
// @Produce  json
// @Success  200 {object} response.Envelope{data=[]review.Review}
// @Router   /v1/admin/reviews [get]
func (s *Service) AdminList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT rv.id::text, rv.rating, rv.body, rv.photos,
		       COALESCE(u.name, u.phone, ''), rv.created_at, rv.target_type, rv.hidden,
		       COALESCE((SELECT slug FROM destinations WHERE id = rv.target_id),
		                (SELECT slug FROM treks WHERE id = rv.target_id), '')
		FROM reviews rv LEFT JOIN users u ON u.id = rv.user_id
		ORDER BY rv.created_at DESC
		LIMIT 200`)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()

	out := make([]Review, 0)
	for rows.Next() {
		var rv Review
		var created any
		if err := rows.Scan(&rv.ID, &rv.Rating, &rv.Body, &rv.Photos, &rv.Author, &created,
			&rv.TargetType, &rv.Hidden, &rv.TargetSlug); err != nil {
			response.Internal(w, err)
			return
		}
		rv.CreatedAt = toString(created)
		out = append(out, rv)
	}
	response.OK(w, out)
}

// AdminDelete godoc
// @Summary  Delete any review (admin)
// @Tags     admin-reviews
// @Security BearerAuth
// @Param    id path string true "Review ID"
// @Success  204
// @Failure  404 {object} response.Envelope
// @Router   /v1/admin/reviews/{id} [delete]
func (s *Service) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var targetType, targetID string
	err := s.pool.QueryRow(r.Context(),
		`SELECT target_type, target_id::text FROM reviews WHERE id = $1::uuid`, id).Scan(&targetType, &targetID)
	if err != nil {
		response.NotFound(w, "review not found")
		return
	}
	if _, err := s.pool.Exec(r.Context(), `DELETE FROM reviews WHERE id = $1::uuid`, id); err != nil {
		response.Internal(w, err)
		return
	}
	s.recompute(r.Context(), targetType, targetID)
	response.NoContent(w)
}

func toString(v any) string {
	if t, ok := v.(interface{ Format(string) string }); ok {
		return t.Format("2006-01-02T15:04:05Z07:00")
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
