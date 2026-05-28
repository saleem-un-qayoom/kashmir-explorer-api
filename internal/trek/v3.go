// Package trek — V3 add-ons: track recordings (saved GPX hikes)
// and summit completions ("bagged" peaks / treks for the badge wall).
//
//   POST /v1/tracks               · save a recording
//   GET  /v1/me/tracks            · my recordings (most-recent first)
//   GET  /v1/tracks/share/{token} · public share-link viewer
//   POST /v1/treks/{slug}/bag     · mark a trek completed
//   GET  /v1/me/completions       · my summit log (for the badge wall)
//   GET  /v1/admin/tracks         · admin: all recordings

package trek

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	mw "github.com/kashmir-explorer/api/internal/middleware"
	"github.com/kashmir-explorer/api/pkg/response"
)

type V3 struct{ pool *pgxpool.Pool }

func NewV3(pool *pgxpool.Pool) *V3 { return &V3{pool: pool} }

/* ── tracks ─────────────────────────────────────────────── */

type createTrackReq struct {
	TrekSlug     string          `json:"trek_slug,omitempty"`
	Name         string          `json:"name"`
	StartedAt    string          `json:"started_at"`
	EndedAt      string          `json:"ended_at"`
	DistanceM    int             `json:"distance_m"`
	DurationS    int             `json:"duration_s"`
	GainM        int             `json:"gain_m"`
	LossM        int             `json:"loss_m"`
	MaxAltitudeM *int            `json:"max_altitude_m,omitempty"`
	Polyline     json.RawMessage `json:"polyline"`
	IsPublic     bool            `json:"is_public"`
}

func (s *V3) CreateTrack(w http.ResponseWriter, r *http.Request) {
	userID := mw.UserID(r)
	if userID == "" { response.Unauthorized(w, "login required"); return }
	var body createTrackReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "invalid body"); return
	}
	if body.Name == "" { body.Name = "Untitled hike" }
	if body.Polyline == nil || len(body.Polyline) == 0 {
		body.Polyline = json.RawMessage("[]")
	}

	var trekID *string
	if body.TrekSlug != "" {
		var id string
		if err := s.pool.QueryRow(r.Context(),
			`SELECT id::text FROM treks WHERE slug = $1`, body.TrekSlug,
		).Scan(&id); err == nil {
			trekID = &id
		}
	}

	var shareToken *string
	if body.IsPublic {
		t := randHex(10)
		shareToken = &t
	}

	var id string
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO track_recordings
		  (user_id, trek_id, name, started_at, ended_at,
		   distance_m, duration_s, gain_m, loss_m, max_altitude_m,
		   polyline, share_token, is_public)
		VALUES ($1, $2::uuid, $3, $4::timestamptz, NULLIF($5,'')::timestamptz,
		        $6, $7, $8, $9, $10, $11::jsonb, $12, $13)
		RETURNING id::text
	`, userID, trekID, body.Name, body.StartedAt, body.EndedAt,
		body.DistanceM, body.DurationS, body.GainM, body.LossM, body.MaxAltitudeM,
		string(body.Polyline), shareToken, body.IsPublic).Scan(&id)
	if err != nil { response.Internal(w, err); return }

	response.Created(w, map[string]any{
		"id": id, "share_token": shareToken,
	})
}

func (s *V3) MyTracks(w http.ResponseWriter, r *http.Request) {
	userID := mw.UserID(r)
	if userID == "" { response.Unauthorized(w, "login required"); return }

	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, name, started_at, ended_at,
		       distance_m, duration_s, gain_m, max_altitude_m,
		       (SELECT slug FROM treks WHERE treks.id = tr.trek_id) AS trek_slug,
		       share_token, is_public, created_at
		FROM track_recordings tr
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 100
	`, userID)
	if err != nil { response.Internal(w, err); return }
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id, name string
		var started, ended, created any
		var dist, dur, gain int
		var maxAlt *int
		var trekSlug, share *string
		var isPub bool
		_ = rows.Scan(&id, &name, &started, &ended, &dist, &dur, &gain, &maxAlt,
			&trekSlug, &share, &isPub, &created)
		out = append(out, map[string]any{
			"id": id, "name": name, "started_at": started, "ended_at": ended,
			"distance_m": dist, "duration_s": dur, "gain_m": gain,
			"max_altitude_m": maxAlt, "trek_slug": trekSlug,
			"share_token": share, "is_public": isPub, "created_at": created,
		})
	}
	response.OK(w, out)
}

func (s *V3) ShareTrack(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	var id, name string
	var started, ended, polyline any
	var dist, dur, gain int
	var maxAlt *int
	var trekSlug *string

	err := s.pool.QueryRow(r.Context(), `
		SELECT tr.id::text, tr.name, tr.started_at, tr.ended_at,
		       tr.distance_m, tr.duration_s, tr.gain_m, tr.max_altitude_m,
		       tr.polyline,
		       (SELECT slug FROM treks WHERE treks.id = tr.trek_id)
		FROM track_recordings tr
		WHERE tr.share_token = $1 AND tr.is_public = true
	`, token).Scan(&id, &name, &started, &ended, &dist, &dur, &gain, &maxAlt,
		&polyline, &trekSlug)
	if err != nil { response.NotFound(w, "track not found"); return }

	response.OK(w, map[string]any{
		"id": id, "name": name, "started_at": started, "ended_at": ended,
		"distance_m": dist, "duration_s": dur, "gain_m": gain,
		"max_altitude_m": maxAlt, "trek_slug": trekSlug,
		"polyline": polyline,
	})
}

/* ── completions / bag ──────────────────────────────────── */

type bagReq struct {
	Notes            string `json:"notes,omitempty"`
	TrackRecordingID string `json:"track_recording_id,omitempty"`
}

func (s *V3) Bag(w http.ResponseWriter, r *http.Request) {
	userID := mw.UserID(r)
	if userID == "" { response.Unauthorized(w, "login required"); return }
	slug := chi.URLParam(r, "slug")

	var body bagReq
	_ = json.NewDecoder(r.Body).Decode(&body)

	var trekID string
	if err := s.pool.QueryRow(r.Context(),
		`SELECT id::text FROM treks WHERE slug = $1`, slug,
	).Scan(&trekID); err != nil {
		response.NotFound(w, "trek not found"); return
	}

	var id string
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO summit_completions (user_id, trek_id, notes, track_recording_id)
		VALUES ($1, $2::uuid, NULLIF($3,''), NULLIF($4,'')::uuid)
		ON CONFLICT (user_id, trek_id) WHERE trek_id IS NOT NULL
		DO UPDATE SET notes = EXCLUDED.notes,
		              track_recording_id = COALESCE(EXCLUDED.track_recording_id, summit_completions.track_recording_id),
		              completed_at = now()
		RETURNING id::text
	`, userID, trekID, body.Notes, body.TrackRecordingID).Scan(&id)
	if err != nil { response.Internal(w, err); return }

	response.Created(w, map[string]any{"id": id, "trek_slug": slug})
}

func (s *V3) MyCompletions(w http.ResponseWriter, r *http.Request) {
	userID := mw.UserID(r)
	if userID == "" { response.Unauthorized(w, "login required"); return }

	rows, err := s.pool.Query(r.Context(), `
		SELECT c.id::text, c.completed_at, c.notes,
		       t.slug, t.name, t.max_altitude_m, t.difficulty,
		       c.track_recording_id::text
		FROM summit_completions c
		LEFT JOIN treks t ON t.id = c.trek_id
		WHERE c.user_id = $1 AND c.trek_id IS NOT NULL
		ORDER BY c.completed_at DESC
	`, userID)
	if err != nil { response.Internal(w, err); return }
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id, slug, name, diff string
		var notes, trackID *string
		var completedAt any
		var maxAlt *int
		_ = rows.Scan(&id, &completedAt, &notes, &slug, &name, &maxAlt, &diff, &trackID)
		out = append(out, map[string]any{
			"id": id, "completed_at": completedAt, "notes": notes,
			"trek_slug": slug, "trek_name": name, "max_altitude_m": maxAlt,
			"difficulty": diff, "track_recording_id": trackID,
		})
	}
	response.OK(w, out)
}

/* ── admin ──────────────────────────────────────────────── */

func (s *V3) AdminTracks(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT tr.id::text, tr.name, COALESCE(u.name, u.phone, '—'),
		       tr.distance_m, tr.duration_s, tr.gain_m, tr.max_altitude_m,
		       tr.is_public, tr.created_at,
		       (SELECT slug FROM treks WHERE treks.id = tr.trek_id)
		FROM track_recordings tr
		LEFT JOIN users u ON u.id = tr.user_id
		ORDER BY tr.created_at DESC
		LIMIT 200
	`)
	if err != nil { response.Internal(w, err); return }
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id, name, user string
		var dist, dur, gain int
		var maxAlt *int
		var pub bool
		var created any
		var slug *string
		_ = rows.Scan(&id, &name, &user, &dist, &dur, &gain, &maxAlt, &pub, &created, &slug)
		out = append(out, map[string]any{
			"id": id, "name": name, "user": user,
			"distance_m": dist, "duration_s": dur, "gain_m": gain,
			"max_altitude_m": maxAlt, "is_public": pub,
			"created_at": created, "trek_slug": slug,
		})
	}
	response.OK(w, out)
}

/* ── helpers ────────────────────────────────────────────── */

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
