// Package report — crowdsourced trek issue reports.
//
//   POST /v1/treks/{slug}/report      · user files an issue
//   GET  /v1/admin/reports            · admin queue
//   POST /v1/admin/reports/{id}/resolve · admin closes
package report

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	mw "github.com/kashmir-explorer/api/internal/middleware"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

type createReq struct {
	Category    string  `json:"category"`     // V2: wrong_path | blocked | unsafe | wildlife | other
	                                          // V3: snow | trail | water
	Body        string  `json:"body"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	Severity    int     `json:"severity"`     // V3 · 1..5 chinar-leaf dial
	PhotoURL    string  `json:"photo_url"`    // V3 · optional R2 upload
	WaypointIdx *int    `json:"waypoint_idx"` // V3 · 0-based on the trek polyline
}

// POST /v1/treks/{slug}/report
func (s *Service) Create(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	userID := mw.UserID(r)
	var body createReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid body"); return
	}
	if body.Category == "" {
		response.BadRequest(w, "category required"); return
	}

	var trekID string
	if err := s.pool.QueryRow(r.Context(),
		`SELECT id::text FROM treks WHERE slug = $1`, slug,
	).Scan(&trekID); err != nil {
		response.NotFound(w, "trek not found"); return
	}

	sev := body.Severity
	if sev < 1 { sev = 3 }
	if sev > 5 { sev = 5 }

	var id string
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO trek_reports
		  (trek_id, user_id, category, body, location,
		   severity, photo_url, waypoint_idx, expires_at)
		VALUES ($1::uuid, $2, $3, $4,
		        CASE WHEN $5 != 0 OR $6 != 0
		             THEN ST_GeogFromText('POINT(' || $6 || ' ' || $5 || ')')
		             ELSE NULL END,
		        $7, NULLIF($8, ''), $9,
		        now() + INTERVAL '14 days')
		RETURNING id::text
	`, trekID, userID, body.Category, body.Body, body.Lat, body.Lng,
		sev, body.PhotoURL, body.WaypointIdx).Scan(&id)
	if err != nil { response.Internal(w, err); return }

	response.Created(w, map[string]any{"id": id, "status": "open", "severity": sev})
}

// GET /v1/treks/{slug}/reports — public feed of unresolved community
// trail-conditions for a trek. Auto-hides expired (>14d).
func (s *Service) PublicList(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	rows, err := s.pool.Query(r.Context(), `
		SELECT r.id::text, r.category, COALESCE(r.severity, 3),
		       r.body, r.photo_url, r.waypoint_idx, r.created_at,
		       COALESCE(u.name, '') AS reporter
		FROM trek_reports r
		JOIN treks t ON t.id = r.trek_id
		LEFT JOIN users u ON u.id = r.user_id
		WHERE t.slug = $1
		  AND r.status IN ('open','reviewing')
		  AND (r.expires_at IS NULL OR r.expires_at > now())
		ORDER BY r.severity DESC NULLS LAST, r.created_at DESC
		LIMIT 50
	`, slug)
	if err != nil { response.Internal(w, err); return }
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id, cat, photo, reporter string
		var body *string
		var photoP, reporterP *string = &photo, &reporter
		var sev int
		var wpIdx *int
		var createdAt any
		_ = rows.Scan(&id, &cat, &sev, &body, &photoP, &wpIdx, &createdAt, &reporterP)
		out = append(out, map[string]any{
			"id": id, "category": cat, "severity": sev,
			"body": body, "photo_url": photoP, "waypoint_idx": wpIdx,
			"created_at": createdAt, "reporter": reporterP,
		})
	}
	response.OK(w, out)
}

// GET /v1/admin/reports
func (s *Service) AdminList(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" { status = "open" }

	rows, err := s.pool.Query(r.Context(), `
		SELECT r.id::text, r.category, r.body, r.status, r.created_at,
		       t.slug AS trek_slug, t.name AS trek_name,
		       COALESCE(u.name, u.phone, ''),
		       ST_X(r.location::geometry), ST_Y(r.location::geometry),
		       COALESCE(r.severity, 3), r.photo_url, r.waypoint_idx, r.expires_at
		FROM trek_reports r
		JOIN treks t ON t.id = r.trek_id
		LEFT JOIN users u ON u.id = r.user_id
		WHERE r.status = $1
		ORDER BY r.severity DESC NULLS LAST, r.created_at DESC
	`, status)
	if err != nil { response.Internal(w, err); return }
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id, cat, st, slug, name, reporter string
		var body, photoURL *string
		var createdAt, expiresAt any
		var lng, lat *float64
		var sev int
		var wpIdx *int
		_ = rows.Scan(&id, &cat, &body, &st, &createdAt, &slug, &name, &reporter, &lng, &lat,
			&sev, &photoURL, &wpIdx, &expiresAt)
		out = append(out, map[string]any{
			"id": id, "category": cat, "body": body, "status": st,
			"created_at": createdAt, "trek_slug": slug, "trek_name": name,
			"reporter": reporter, "lat": lat, "lng": lng,
			"severity": sev, "photo_url": photoURL, "waypoint_idx": wpIdx,
			"expires_at": expiresAt,
		})
	}
	response.OK(w, out)
}

type resolveReq struct {
	Status    string `json:"status"`     // resolved | dismissed
	AdminNote string `json:"admin_note"`
}

// POST /v1/admin/reports/{id}/resolve
func (s *Service) AdminResolve(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body resolveReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid body"); return
	}
	if body.Status == "" { body.Status = "resolved" }
	if _, err := s.pool.Exec(r.Context(), `
		UPDATE trek_reports SET status = $2, admin_note = $3, resolved_at = now()
		WHERE id = $1::uuid
	`, id, body.Status, body.AdminNote); err != nil {
		response.Internal(w, err); return
	}
	response.OK(w, map[string]string{"status": body.Status})
}
