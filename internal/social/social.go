// Package social — the social graph: follows, public profiles, activity feed.
package social

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	mw "github.com/kashmir-explorer/api/internal/middleware"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

// Profile is a user's public profile (no PII — name + avatar + aggregates).
type Profile struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
	JoinedAt    string  `json:"joined_at"`
	Followers   int     `json:"followers"`
	Following   int     `json:"following"`
	Completions int     `json:"completions"`
	Reviews     int     `json:"reviews"`
}

// UserCard is the compact form used in follower/following lists.
type UserCard struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

// FeedItem is one activity-feed entry — a review or a summit completion.
type FeedItem struct {
	Type       string  `json:"type"` // 'review' | 'completion'
	UserID     string  `json:"user_id"`
	UserName   string  `json:"user_name"`
	AvatarURL  *string `json:"avatar_url,omitempty"`
	TargetType string  `json:"target_type"`
	TargetSlug string  `json:"target_slug"`
	TargetName string  `json:"target_name"`
	Rating     *int    `json:"rating,omitempty"`
	Body       *string `json:"body,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

// ─── Follow / unfollow ────────────────────────────────────────

// Follow godoc
// @Summary  Follow a user
// @Tags     social
// @Security BearerAuth
// @Param    id path string true "User ID to follow"
// @Success  204
// @Failure  400 {object} response.Envelope
// @Router   /v1/users/{id}/follow [post]
func (s *Service) Follow(w http.ResponseWriter, r *http.Request) {
	me := mw.UserID(r)
	target := chi.URLParam(r, "id")
	if me == "" {
		response.Unauthorized(w, "login required")
		return
	}
	if me == target {
		response.BadRequest(w, "cannot follow yourself")
		return
	}
	if _, err := s.pool.Exec(r.Context(),
		`INSERT INTO follows (follower_id, following_id) VALUES ($1, $2::uuid) ON CONFLICT DO NOTHING`,
		me, target); err != nil {
		response.Internal(w, err)
		return
	}
	response.NoContent(w)
}

// Unfollow godoc
// @Summary  Unfollow a user
// @Tags     social
// @Security BearerAuth
// @Param    id path string true "User ID to unfollow"
// @Success  204
// @Router   /v1/users/{id}/follow [delete]
func (s *Service) Unfollow(w http.ResponseWriter, r *http.Request) {
	me := mw.UserID(r)
	target := chi.URLParam(r, "id")
	if _, err := s.pool.Exec(r.Context(),
		`DELETE FROM follows WHERE follower_id = $1 AND following_id = $2::uuid`, me, target); err != nil {
		response.Internal(w, err)
		return
	}
	response.NoContent(w)
}

// ─── Profile + follower lists ─────────────────────────────────

// Profile godoc
// @Summary  Public profile (name, avatar, aggregate counts)
// @Tags     social
// @Produce  json
// @Param    id path string true "User ID"
// @Success  200 {object} response.Envelope{data=social.Profile}
// @Failure  404 {object} response.Envelope
// @Router   /v1/users/{id} [get]
func (s *Service) Profile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var p Profile
	var joined any
	err := s.pool.QueryRow(r.Context(), `
		SELECT u.id::text, COALESCE(u.name, 'Traveller'), u.avatar_url, u.created_at,
		       (SELECT COUNT(*) FROM follows WHERE following_id = u.id),
		       (SELECT COUNT(*) FROM follows WHERE follower_id = u.id),
		       (SELECT COUNT(*) FROM summit_completions WHERE user_id = u.id AND trek_id IS NOT NULL),
		       (SELECT COUNT(*) FROM reviews WHERE user_id = u.id AND hidden = false)
		FROM users u WHERE u.id = $1::uuid`, id).
		Scan(&p.ID, &p.Name, &p.AvatarURL, &joined, &p.Followers, &p.Following, &p.Completions, &p.Reviews)
	if err != nil {
		response.NotFound(w, "user not found")
		return
	}
	p.JoinedAt = toString(joined)
	response.OK(w, p)
}

// Followers godoc
// @Summary  Users who follow this user
// @Tags     social
// @Produce  json
// @Param    id path string true "User ID"
// @Success  200 {object} response.Envelope{data=[]social.UserCard}
// @Router   /v1/users/{id}/followers [get]
func (s *Service) Followers(w http.ResponseWriter, r *http.Request) {
	s.cards(w, r, `
		SELECT u.id::text, COALESCE(u.name, 'Traveller'), u.avatar_url
		FROM follows f JOIN users u ON u.id = f.follower_id
		WHERE f.following_id = $1::uuid ORDER BY f.created_at DESC LIMIT 200`)
}

// Following godoc
// @Summary  Users this user follows
// @Tags     social
// @Produce  json
// @Param    id path string true "User ID"
// @Success  200 {object} response.Envelope{data=[]social.UserCard}
// @Router   /v1/users/{id}/following [get]
func (s *Service) Following(w http.ResponseWriter, r *http.Request) {
	s.cards(w, r, `
		SELECT u.id::text, COALESCE(u.name, 'Traveller'), u.avatar_url
		FROM follows f JOIN users u ON u.id = f.following_id
		WHERE f.follower_id = $1::uuid ORDER BY f.created_at DESC LIMIT 200`)
}

func (s *Service) cards(w http.ResponseWriter, r *http.Request, query string) {
	id := chi.URLParam(r, "id")
	rows, err := s.pool.Query(r.Context(), query, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()
	out := make([]UserCard, 0)
	for rows.Next() {
		var c UserCard
		if err := rows.Scan(&c.ID, &c.Name, &c.AvatarURL); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, c)
	}
	response.OK(w, out)
}

// ─── Activity feed ────────────────────────────────────────────

// Feed godoc
// @Summary  Activity feed — reviews & completions from people you follow
// @Tags     social
// @Security BearerAuth
// @Produce  json
// @Success  200 {object} response.Envelope{data=[]social.FeedItem}
// @Router   /v1/me/feed [get]
func (s *Service) Feed(w http.ResponseWriter, r *http.Request) {
	me := mw.UserID(r)
	if me == "" {
		response.Unauthorized(w, "login required")
		return
	}
	rows, err := s.pool.Query(r.Context(), `
		SELECT 'review' AS type, rv.user_id::text, COALESCE(u.name, 'Traveller'), u.avatar_url,
		       rv.target_type,
		       COALESCE((SELECT slug FROM destinations WHERE id = rv.target_id),
		                (SELECT slug FROM treks WHERE id = rv.target_id), '') AS target_slug,
		       COALESCE((SELECT name FROM destinations WHERE id = rv.target_id),
		                (SELECT name FROM treks WHERE id = rv.target_id), '') AS target_name,
		       rv.rating, rv.body, rv.created_at
		FROM reviews rv JOIN users u ON u.id = rv.user_id
		WHERE rv.user_id IN (SELECT following_id FROM follows WHERE follower_id = $1) AND rv.hidden = false
		UNION ALL
		SELECT 'completion', c.user_id::text, COALESCE(u.name, 'Traveller'), u.avatar_url,
		       'trek', t.slug, t.name, NULL::int, c.notes, c.completed_at
		FROM summit_completions c JOIN users u ON u.id = c.user_id LEFT JOIN treks t ON t.id = c.trek_id
		WHERE c.user_id IN (SELECT following_id FROM follows WHERE follower_id = $1) AND c.trek_id IS NOT NULL
		ORDER BY 10 DESC
		LIMIT 50`, me)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()
	out := make([]FeedItem, 0)
	for rows.Next() {
		var it FeedItem
		var created any
		if err := rows.Scan(&it.Type, &it.UserID, &it.UserName, &it.AvatarURL, &it.TargetType,
			&it.TargetSlug, &it.TargetName, &it.Rating, &it.Body, &created); err != nil {
			response.Internal(w, err)
			return
		}
		it.CreatedAt = toString(created)
		out = append(out, it)
	}
	response.OK(w, out)
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
