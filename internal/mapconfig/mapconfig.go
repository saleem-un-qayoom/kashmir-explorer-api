// Package mapconfig — app-wide map relief configuration.
//
// A single row (map_config) holds the shared terrain-exaggeration factor the
// mobile app applies to its native MapLibre hillshade layer. Engine selection
// (Cesium vs Mapbox) was removed — the app now uses native MapLibre across
// every map surface. The admin edits the exaggeration; the mobile app reads
// it at launch and applies it via the hillshade / contour layer paint.
package mapconfig

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

// Config is the public/admin map-config payload.
type Config struct {
	// TerrainExaggeration scales terrain relief (0–3).
	TerrainExaggeration float64   `json:"terrain_exaggeration"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ConfigInput is the admin update body.
type ConfigInput struct {
	TerrainExaggeration float64 `json:"terrain_exaggeration"`
}

func (s *Service) read(r *http.Request) (Config, error) {
	var c Config
	err := s.pool.QueryRow(r.Context(),
		`SELECT terrain_exaggeration, updated_at FROM map_config WHERE id = 1`).
		Scan(&c.TerrainExaggeration, &c.UpdatedAt)
	return c, err
}

// Get godoc
// @Summary  Get the active map configuration
// @Tags     map-config
// @Produce  json
// @Success  200 {object} response.Envelope{data=mapconfig.Config}
// @Router   /v1/map-config [get]
func (s *Service) Get(w http.ResponseWriter, r *http.Request) {
	c, err := s.read(r)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, c)
}

// AdminGet godoc
// @Summary  Get the map configuration (admin)
// @Tags     admin-map-config
// @Security BearerAuth
// @Produce  json
// @Success  200 {object} response.Envelope{data=mapconfig.Config}
// @Router   /v1/admin/map-config [get]
func (s *Service) AdminGet(w http.ResponseWriter, r *http.Request) { s.Get(w, r) }

// AdminUpdate godoc
// @Summary  Update the map configuration (admin)
// @Tags     admin-map-config
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body mapconfig.ConfigInput true "Map config"
// @Success  200 {object} response.Envelope{data=mapconfig.Config}
// @Failure  400 {object} response.Envelope
// @Router   /v1/admin/map-config [put]
func (s *Service) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	var in ConfigInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	if in.TerrainExaggeration < 0 || in.TerrainExaggeration > 3 {
		response.BadRequest(w, "terrain_exaggeration must be between 0 and 3")
		return
	}
	_, err := s.pool.Exec(r.Context(), `
		UPDATE map_config
		SET terrain_exaggeration = $1, updated_at = now()
		WHERE id = 1
	`, in.TerrainExaggeration)
	if err != nil {
		response.Internal(w, err)
		return
	}
	c, err := s.read(r)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, c)
}
