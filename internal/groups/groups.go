// Package groups — trip group rooms for live location-share between friends.
//
//	POST   /v1/groups               · create + return invite code
//	POST   /v1/groups/join          · join via invite code
//	GET    /v1/groups/{code}        · meta + member list
//	DELETE /v1/groups/{code}/leave  · leave
//
// Live position broadcasts go over WS /ws/group/{code}.
package groups

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	mw "github.com/kashmir-explorer/api/internal/middleware"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

type createReq struct {
	Name     string `json:"name"`
	TrekSlug string `json:"trek_slug,omitempty"`
}

// Group doc-models (OpenAPI/codegen).
type GroupCreateInput struct {
	Name     string `json:"name"`
	TrekSlug string `json:"trek_slug,omitempty"`
}
type GroupJoinInput struct {
	Code string `json:"code"`
}
type GroupMember struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}
type Group struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	InviteCode string        `json:"invite_code"`
	TrekSlug   *string       `json:"trek_slug,omitempty"`
	Members    []GroupMember `json:"members,omitempty"`
}

// Create godoc
// @Summary  Create a trip group (returns invite code)
// @Tags     groups
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body groups.GroupCreateInput true "Group"
// @Success  201 {object} response.Envelope{data=groups.Group}
// @Router   /v1/groups [post]
func (s *Service) Create(w http.ResponseWriter, r *http.Request) {
	userID := mw.UserID(r)
	var body createReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	if body.Name == "" {
		body.Name = "My group"
	}

	code := newInviteCode()
	var id string
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO trip_groups (owner_id, name, invite_code, trek_slug)
		VALUES ($1, $2, $3, NULLIF($4, ''))
		RETURNING id::text
	`, userID, body.Name, code, body.TrekSlug).Scan(&id)
	if err != nil {
		response.Internal(w, err)
		return
	}

	_, _ = s.pool.Exec(r.Context(),
		`INSERT INTO trip_group_members (group_id, user_id) VALUES ($1::uuid, $2)`,
		id, userID)

	response.Created(w, map[string]any{
		"id": id, "name": body.Name, "invite_code": code, "trek_slug": body.TrekSlug,
	})
}

type joinReq struct {
	Code string `json:"code"`
}

// Join godoc
// @Summary  Join a trip group by invite code
// @Tags     groups
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body groups.GroupJoinInput true "Invite code"
// @Success  200 {object} response.Envelope
// @Failure  404 {object} response.Envelope
// @Router   /v1/groups/join [post]
func (s *Service) Join(w http.ResponseWriter, r *http.Request) {
	userID := mw.UserID(r)
	var body joinReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		response.BadRequest(w, "code required")
		return
	}
	var gid string
	err := s.pool.QueryRow(r.Context(),
		`SELECT id::text FROM trip_groups WHERE invite_code = $1 AND expires_at > now()`,
		strings.ToUpper(body.Code),
	).Scan(&gid)
	if err != nil {
		response.NotFound(w, "invite code invalid or expired")
		return
	}

	_, err = s.pool.Exec(r.Context(),
		`INSERT INTO trip_group_members (group_id, user_id) VALUES ($1::uuid, $2)
		 ON CONFLICT DO NOTHING`,
		gid, userID)
	if err != nil {
		response.Internal(w, err)
		return
	}

	response.OK(w, map[string]any{"group_id": gid})
}

// Get godoc
// @Summary  Get a group's members + invite code
// @Tags     groups
// @Security BearerAuth
// @Produce  json
// @Param    code path string true "Invite code"
// @Success  200 {object} response.Envelope{data=groups.Group}
// @Failure  403 {object} response.Envelope
// @Failure  404 {object} response.Envelope
// @Router   /v1/groups/{code} [get]
func (s *Service) Get(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(chi.URLParam(r, "code"))
	userID := mw.UserID(r)

	var id, name string
	var trekSlug *string
	err := s.pool.QueryRow(r.Context(),
		`SELECT id::text, name, trek_slug FROM trip_groups WHERE invite_code = $1`, code,
	).Scan(&id, &name, &trekSlug)
	if err != nil {
		response.NotFound(w, "group not found")
		return
	}

	// Must be a member.
	var isMember bool
	_ = s.pool.QueryRow(r.Context(),
		`SELECT EXISTS (SELECT 1 FROM trip_group_members WHERE group_id = $1::uuid AND user_id = $2)`,
		id, userID).Scan(&isMember)
	if !isMember {
		response.Forbidden(w, "not a member of this group")
		return
	}

	rows, err := s.pool.Query(r.Context(), `
		SELECT u.id::text, COALESCE(u.name, u.phone, 'Anon') FROM trip_group_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.group_id = $1::uuid
	`, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()

	members := []map[string]string{}
	for rows.Next() {
		var uid, label string
		if err := rows.Scan(&uid, &label); err != nil {
			response.Internal(w, err)
			return
		}
		members = append(members, map[string]string{"id": uid, "label": label})
	}

	response.OK(w, map[string]any{
		"id":          id,
		"name":        name,
		"invite_code": code,
		"trek_slug":   trekSlug,
		"members":     members,
	})
}

// Leave godoc
// @Summary  Leave a trip group
// @Tags     groups
// @Security BearerAuth
// @Param    code path string true "Invite code"
// @Success  204
// @Router   /v1/groups/{code}/leave [delete]
func (s *Service) Leave(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(chi.URLParam(r, "code"))
	userID := mw.UserID(r)
	_, _ = s.pool.Exec(r.Context(), `
		DELETE FROM trip_group_members
		WHERE user_id = $1
		  AND group_id = (SELECT id FROM trip_groups WHERE invite_code = $2)
	`, userID, code)
	response.NoContent(w)
}

func newInviteCode() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
	return strings.ToUpper(enc)[:6]
}
