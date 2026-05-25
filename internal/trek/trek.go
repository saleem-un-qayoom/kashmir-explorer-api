// Package trek — trek listing + detail.
package trek

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct{ pool *pgxpool.Pool }

func NewService(pool *pgxpool.Pool) *Service { return &Service{pool: pool} }

type Trek struct {
	ID            string  `json:"id"`
	Slug          string  `json:"slug"`
	Name          string  `json:"name"`
	Difficulty    string  `json:"difficulty"`
	TrekType      string  `json:"trek_type"`
	DurationDays  int     `json:"duration_days"`
	DistanceKm    float64 `json:"distance_km"`
	MaxAltitudeM  int     `json:"max_altitude_m"`
	StartPoint    *string `json:"start_point,omitempty"`
	EndPoint      *string `json:"end_point,omitempty"`
	BestMonths    []int   `json:"best_months,omitempty"`
	Permits       []string `json:"permits,omitempty"`
	AmsRisk       bool    `json:"ams_risk"`
	Status        string  `json:"status"`
	ClosureReason *string `json:"closure_reason,omitempty"`
	Tagline       *string `json:"tagline,omitempty"`
	Uniqueness    *string `json:"uniqueness,omitempty"`
	Rating        float64 `json:"rating"`
	ReviewCount   int     `json:"review_count"`
	GuideAvailable bool   `json:"guide_available"`
	GuidePriceINR int     `json:"guide_price_inr"`
}

// GET /v1/treks ?difficulty=&duration=&type=&open=
func (s *Service) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	difficulty := q.Get("difficulty")
	openOnly := q.Get("open") == "true"
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit == 0 {
		limit = 50
	}

	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, slug, name, difficulty, trek_type, duration_days, distance_km,
		       max_altitude_m, start_point, end_point, best_months, permits,
		       ams_risk, status, closure_reason, tagline, uniqueness, rating, review_count,
		       guide_available, guide_price_inr
		FROM treks
		WHERE is_published = true
		  AND ($1 = '' OR difficulty = $1)
		  AND (NOT $2 OR status = 'open')
		ORDER BY rating DESC
		LIMIT $3
	`, difficulty, openOnly, limit)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()
	out := make([]Trek, 0)
	for rows.Next() {
		var t Trek
		if err := rows.Scan(
			&t.ID, &t.Slug, &t.Name, &t.Difficulty, &t.TrekType, &t.DurationDays, &t.DistanceKm,
			&t.MaxAltitudeM, &t.StartPoint, &t.EndPoint, &t.BestMonths, &t.Permits,
			&t.AmsRisk, &t.Status, &t.ClosureReason, &t.Tagline, &t.Uniqueness, &t.Rating, &t.ReviewCount,
			&t.GuideAvailable, &t.GuidePriceINR,
		); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, t)
	}
	response.OK(w, out)
}

// GET /v1/treks/{slug}
func (s *Service) Get(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	var t Trek
	var waypoints, gearList any
	err := s.pool.QueryRow(r.Context(), `
		SELECT id::text, slug, name, difficulty, trek_type, duration_days, distance_km,
		       max_altitude_m, start_point, end_point, best_months, permits,
		       ams_risk, status, closure_reason, tagline, uniqueness, rating, review_count,
		       guide_available, guide_price_inr, waypoints, gear_list
		FROM treks WHERE slug = $1 AND is_published = true
	`, slug).Scan(
		&t.ID, &t.Slug, &t.Name, &t.Difficulty, &t.TrekType, &t.DurationDays, &t.DistanceKm,
		&t.MaxAltitudeM, &t.StartPoint, &t.EndPoint, &t.BestMonths, &t.Permits,
		&t.AmsRisk, &t.Status, &t.ClosureReason, &t.Tagline, &t.Uniqueness, &t.Rating, &t.ReviewCount,
		&t.GuideAvailable, &t.GuidePriceINR, &waypoints, &gearList,
	)
	if err != nil {
		response.NotFound(w, "trek not found")
		return
	}
	response.OK(w, map[string]any{
		"trek":      t,
		"waypoints": waypoints,
		"gear_list": gearList,
	})
}

// GET /v1/treks/{slug}/path — densified polyline + waypoints for offline nav.
func (s *Service) Path(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var polyJSON, waypointsJSON []byte
	var distKm float64
	err := s.pool.QueryRow(r.Context(), `
		SELECT
		  COALESCE(path_geojson::text, '[]'),
		  COALESCE(waypoint_coords::text, '[]'),
		  COALESCE(distance_km, 0)
		FROM treks WHERE slug = $1 AND is_published = true
	`, slug).Scan(&polyJSON, &waypointsJSON, &distKm)
	if err != nil {
		response.NotFound(w, "trek path not found")
		return
	}
	if len(polyJSON) <= 2 {
		response.NotFound(w, "trek path not yet digitised")
		return
	}
	response.OK(w, map[string]any{
		"polyline":         json.RawMessage(polyJSON),
		"waypoints":        json.RawMessage(waypointsJSON),
		"total_distance_m": int(distKm * 1000),
		"version":          "1",
	})
}

func (s *Service) AdminCreate(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]string{"todo": "trek create"})
}
func (s *Service) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]string{"todo": "trek update"})
}
