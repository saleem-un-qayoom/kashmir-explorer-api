// Package theme — app-wide color theme overrides.
//
// A single row (app_theme) holds a JSON map of design-token color keys to hex
// values. The mobile app fetches the active theme at launch and applies these
// overrides on top of its built-in palette; the admin edits them.
package theme

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

// Theme is the public/admin theme payload.
type Theme struct {
	// Colors maps design-token keys (e.g. "saffron", "dalBlue") to hex strings.
	Colors    map[string]string `json:"colors"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// ThemeInput is the admin update body (OpenAPI/codegen model).
type ThemeInput struct {
	Colors map[string]string `json:"colors"`
}

// allowedKeys are the design-token color slots the app knows how to override.
// Anything outside this set is rejected so the admin can't inject arbitrary
// keys that the mobile app would silently ignore.
var allowedKeys = map[string]bool{
	"saffron": true, "saffronDeep": true,
	"dalBlue": true, "dalDeep": true,
	"chinarRed": true, "chinarAmber": true,
	"pashmina": true, "pashminaDk": true,
	"sapphire": true, "almond": true, "tulip": true,
	"emerald": true, "mustard": true, "snowMist": true, "terra": true,
	"bg": true, "surface": true, "raised": true,
	"text": true, "text2": true, "text3": true,
	"line": true, "lineStrong": true,
}

func (s *Service) read(r *http.Request) (Theme, error) {
	var (
		raw []byte
		t   Theme
	)
	err := s.pool.QueryRow(r.Context(),
		`SELECT colors, updated_at FROM app_theme WHERE id = 1`).Scan(&raw, &t.UpdatedAt)
	if err != nil {
		return t, err
	}
	if err := json.Unmarshal(raw, &t.Colors); err != nil {
		return t, err
	}
	if t.Colors == nil {
		t.Colors = map[string]string{}
	}
	return t, nil
}

// Get godoc
// @Summary  Get the active app theme (color overrides)
// @Tags     theme
// @Produce  json
// @Success  200 {object} response.Envelope{data=theme.Theme}
// @Router   /v1/theme [get]
func (s *Service) Get(w http.ResponseWriter, r *http.Request) {
	t, err := s.read(r)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, t)
}

// AdminGet godoc
// @Summary  Get the app theme (admin)
// @Tags     admin-theme
// @Security BearerAuth
// @Produce  json
// @Success  200 {object} response.Envelope{data=theme.Theme}
// @Router   /v1/admin/theme [get]
func (s *Service) AdminGet(w http.ResponseWriter, r *http.Request) { s.Get(w, r) }

// AdminUpdate godoc
// @Summary  Replace the app theme color overrides (admin)
// @Tags     admin-theme
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body theme.ThemeInput true "Theme"
// @Success  200 {object} response.Envelope{data=theme.Theme}
// @Failure  400 {object} response.Envelope
// @Router   /v1/admin/theme [put]
func (s *Service) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	var in ThemeInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	clean := map[string]string{}
	for k, v := range in.Colors {
		if !allowedKeys[k] {
			response.BadRequest(w, "unknown color key: "+k)
			return
		}
		if !isHex(v) {
			response.BadRequest(w, "invalid hex for "+k+": "+v)
			return
		}
		clean[k] = v
	}
	raw, err := json.Marshal(clean)
	if err != nil {
		response.Internal(w, err)
		return
	}
	_, err = s.pool.Exec(r.Context(), `
		UPDATE app_theme SET colors = $1, updated_at = now() WHERE id = 1
	`, raw)
	if err != nil {
		response.Internal(w, err)
		return
	}
	t, err := s.read(r)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, t)
}

// isHex accepts #RGB, #RRGGBB, and #RRGGBBAA.
func isHex(v string) bool {
	if len(v) == 0 || v[0] != '#' {
		return false
	}
	h := v[1:]
	if len(h) != 3 && len(h) != 6 && len(h) != 8 {
		return false
	}
	for _, c := range h {
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}
