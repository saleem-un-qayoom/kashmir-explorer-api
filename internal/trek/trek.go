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
	ID            string   `json:"id"`
	Slug          string   `json:"slug"`
	Name          string   `json:"name"`
	Difficulty    *string  `json:"difficulty,omitempty"`
	TrekType      *string  `json:"trek_type,omitempty"`
	DurationDays  *int     `json:"duration_days,omitempty"`
	DistanceKm    *float64 `json:"distance_km,omitempty"`
	MaxAltitudeM  *int     `json:"max_altitude_m,omitempty"`
	StartPoint    *string  `json:"start_point,omitempty"`
	EndPoint      *string  `json:"end_point,omitempty"`
	BestMonths    []int    `json:"best_months,omitempty"`
	Permits       []string `json:"permits,omitempty"`
	AmsRisk       bool     `json:"ams_risk"`
	Status        *string  `json:"status,omitempty"`
	ClosureReason *string `json:"closure_reason,omitempty"`
	Tagline       *string `json:"tagline,omitempty"`
	Uniqueness    *string `json:"uniqueness,omitempty"`
	Rating        float64 `json:"rating"`
	ReviewCount   int     `json:"review_count"`
	GuideAvailable bool   `json:"guide_available"`
	GuidePriceINR *int    `json:"guide_price_inr,omitempty"`
	// AllTrails-style discovery (migration 0009)
	Features       []string `json:"features,omitempty"`
	Activities     []string `json:"activities,omitempty"`
	ElevationGainM *int     `json:"elevation_gain_m,omitempty"`
	RouteType      *string  `json:"route_type,omitempty"`
	HeroImageURL   *string  `json:"hero_image_url,omitempty"` // joined from images
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
		SELECT t.id::text, t.slug, t.name, t.difficulty, t.trek_type, t.duration_days, t.distance_km,
		       t.max_altitude_m, t.start_point, t.end_point, t.best_months, t.permits,
		       t.ams_risk, t.status, t.closure_reason, t.tagline, t.uniqueness, t.rating, t.review_count,
		       t.guide_available, t.guide_price_inr,
		       COALESCE(t.features, '{}'::TEXT[]),
		       COALESCE(t.activities, '{hike}'::TEXT[]),
		       t.elevation_gain_m, t.route_type,
		       (SELECT url FROM images i
		         WHERE i.trek_id = t.id
		         ORDER BY i.is_hero DESC, i.sort_order, i.created_at LIMIT 1)
		FROM treks t
		WHERE t.is_published = true
		  AND ($1 = '' OR t.difficulty = $1)
		  AND (NOT $2 OR t.status = 'open')
		ORDER BY t.rating DESC
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
			&t.Features, &t.Activities, &t.ElevationGainM, &t.RouteType, &t.HeroImageURL,
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
	var waypoints, gearList, sections any
	err := s.pool.QueryRow(r.Context(), `
		SELECT t.id::text, t.slug, t.name, t.difficulty, t.trek_type, t.duration_days, t.distance_km,
		       t.max_altitude_m, t.start_point, t.end_point, t.best_months, t.permits,
		       t.ams_risk, t.status, t.closure_reason, t.tagline, t.uniqueness, t.rating, t.review_count,
		       t.guide_available, t.guide_price_inr, t.waypoints, t.gear_list, t.trail_sections,
		       COALESCE(t.features, '{}'::TEXT[]),
		       COALESCE(t.activities, '{hike}'::TEXT[]),
		       t.elevation_gain_m, t.route_type,
		       (SELECT url FROM images i
		         WHERE i.trek_id = t.id
		         ORDER BY i.is_hero DESC, i.sort_order, i.created_at LIMIT 1)
		FROM treks t WHERE t.slug = $1 AND t.is_published = true
	`, slug).Scan(
		&t.ID, &t.Slug, &t.Name, &t.Difficulty, &t.TrekType, &t.DurationDays, &t.DistanceKm,
		&t.MaxAltitudeM, &t.StartPoint, &t.EndPoint, &t.BestMonths, &t.Permits,
		&t.AmsRisk, &t.Status, &t.ClosureReason, &t.Tagline, &t.Uniqueness, &t.Rating, &t.ReviewCount,
		&t.GuideAvailable, &t.GuidePriceINR, &waypoints, &gearList, &sections,
		&t.Features, &t.Activities, &t.ElevationGainM, &t.RouteType, &t.HeroImageURL,
	)
	if err != nil {
		response.NotFound(w, "trek not found")
		return
	}
	// Flatten the response — the mobile `TrekDetail` type expects every
	// scalar trek field at the root level alongside the JSONB extras.
	flat := map[string]any{
		"id":                     t.ID,
		"slug":                   t.Slug,
		"name":                   t.Name,
		"difficulty":             t.Difficulty,
		"trek_type":              t.TrekType,
		"duration_days":          t.DurationDays,
		"distance_km":            t.DistanceKm,
		"max_altitude_m":         t.MaxAltitudeM,
		"start_point":            t.StartPoint,
		"end_point":              t.EndPoint,
		"best_months":            t.BestMonths,
		"permits":                t.Permits,
		"ams_risk":               t.AmsRisk,
		"status":                 t.Status,
		"closure_reason":         t.ClosureReason,
		"tagline":                t.Tagline,
		"uniqueness":             t.Uniqueness,
		"rating":                 t.Rating,
		"review_count":           t.ReviewCount,
		"guide_available":        t.GuideAvailable,
		"guide_price_inr":        t.GuidePriceINR,
		"features":               t.Features,
		"activities":             t.Activities,
		"elevation_gain_m":       t.ElevationGainM,
		"route_type":             t.RouteType,
		"hero_image_url":         t.HeroImageURL,
		"waypoints":              waypoints,
		"gear_list":              gearList,
		"trail_sections":         sections,
	}
	response.OK(w, flat)
}

// GET /v1/treks/{slug}/path — densified polyline + waypoints for offline nav.
func (s *Service) Path(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var polyJSON, waypointsJSON, phasesJSON []byte
	var distKm float64
	err := s.pool.QueryRow(r.Context(), `
		SELECT
		  COALESCE(path_geojson::text, '[]'),
		  COALESCE(waypoint_coords::text, '[]'),
		  COALESCE(distance_km, 0),
		  COALESCE(path_phases::text, '[]')
		FROM treks WHERE slug = $1 AND is_published = true
	`, slug).Scan(&polyJSON, &waypointsJSON, &distKm, &phasesJSON)
	if err != nil {
		response.NotFound(w, "trek path not found")
		return
	}
	if len(polyJSON) <= 2 && len(phasesJSON) <= 2 {
		response.NotFound(w, "trek path not yet digitised")
		return
	}
	response.OK(w, map[string]any{
		"polyline":         json.RawMessage(polyJSON),
		"waypoints":        json.RawMessage(waypointsJSON),
		"phases":           json.RawMessage(phasesJSON),  // per-day color-coded segments
		"total_distance_m": int(distKm * 1000),
		"version":          "2",
	})
}

// ─── Admin ────────────────────────────────────────────────────

type AdminTrek struct {
	ID              string          `json:"id"`
	Slug            string          `json:"slug"`
	Name            string          `json:"name"`
	Difficulty      string          `json:"difficulty"`
	TrekType        string          `json:"trek_type"`
	DurationDays    int             `json:"duration_days"`
	DistanceKm      *float64        `json:"distance_km"`
	MaxAltitudeM    *int            `json:"max_altitude_m"`
	StartPoint      *string         `json:"start_point"`
	EndPoint        *string         `json:"end_point"`
	BestMonths      []int           `json:"best_months"`
	Permits         []string        `json:"permits"`
	AmsRisk         bool            `json:"ams_risk"`
	Status          string          `json:"status"`
	ClosureReason   *string         `json:"closure_reason"`
	Tagline         *string         `json:"tagline"`
	Uniqueness      *string         `json:"uniqueness"`
	Rating          float64         `json:"rating"`
	ReviewCount     int             `json:"review_count"`
	GuideAvailable  bool            `json:"guide_available"`
	GuidePriceINR   *int            `json:"guide_price_inr"`
	Waypoints       json.RawMessage `json:"waypoints"`
	GearList        json.RawMessage `json:"gear_list"`
	PathGeoJSON     json.RawMessage `json:"path_geojson"`
	PathPhases      json.RawMessage `json:"path_phases"`  // [{day, coordinates: [[lng,lat],…]}]
	WaypointCoords  json.RawMessage `json:"waypoint_coords"`
	TrailSections   json.RawMessage `json:"trail_sections"`
	IsPublished     bool            `json:"is_published"`
	// AllTrails-style discovery (migration 0009)
	Features        []string `json:"features"`
	Activities      []string `json:"activities"`
	ElevationGainM  *int     `json:"elevation_gain_m"`
	RouteType       *string  `json:"route_type"`
}

func (s *Service) AdminCreate(w http.ResponseWriter, r *http.Request) {
	var in AdminTrek
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body"); return
	}
	if in.Name == "" || in.Slug == "" {
		response.BadRequest(w, "name and slug required"); return
	}

	var id string
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO treks
			(slug, name, difficulty, trek_type, duration_days, distance_km,
			 max_altitude_m, start_point, end_point, best_months, permits,
			 ams_risk, status, closure_reason, tagline, uniqueness,
			 guide_available, guide_price_inr, waypoints, gear_list,
			 path_geojson, waypoint_coords, trail_sections, is_published,
			 features, activities, elevation_gain_m, route_type, path_phases)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,
		        $25,$26,$27,$28,$29)
		RETURNING id::text
	`, in.Slug, in.Name, in.Difficulty, in.TrekType, in.DurationDays, in.DistanceKm,
		in.MaxAltitudeM, in.StartPoint, in.EndPoint, in.BestMonths, in.Permits,
		in.AmsRisk, in.Status, in.ClosureReason, in.Tagline, in.Uniqueness,
		in.GuideAvailable, in.GuidePriceINR, in.Waypoints, in.GearList,
		in.PathGeoJSON, in.WaypointCoords, in.TrailSections, in.IsPublished,
		in.Features, in.Activities, in.ElevationGainM, in.RouteType, in.PathPhases,
	).Scan(&id)
	if err != nil {
		response.Internal(w, err); return
	}
	response.Created(w, map[string]string{"id": id})
}

func (s *Service) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in AdminTrek
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body"); return
	}

	_, err := s.pool.Exec(r.Context(), `
		UPDATE treks SET
			slug=$2, name=$3, difficulty=$4, trek_type=$5, duration_days=$6,
			distance_km=$7, max_altitude_m=$8, start_point=$9, end_point=$10,
			best_months=$11, permits=$12, ams_risk=$13, status=$14, closure_reason=$15,
			tagline=$16, uniqueness=$17, guide_available=$18, guide_price_inr=$19,
			waypoints=$20, gear_list=$21, path_geojson=$22, waypoint_coords=$23,
			trail_sections=$24, is_published=$25,
			features=$26, activities=$27, elevation_gain_m=$28, route_type=$29,
			path_phases=$30,
			updated_at=now()
		WHERE id=$1
	`, id, in.Slug, in.Name, in.Difficulty, in.TrekType, in.DurationDays,
		in.DistanceKm, in.MaxAltitudeM, in.StartPoint, in.EndPoint,
		in.BestMonths, in.Permits, in.AmsRisk, in.Status, in.ClosureReason,
		in.Tagline, in.Uniqueness, in.GuideAvailable, in.GuidePriceINR,
		in.Waypoints, in.GearList, in.PathGeoJSON, in.WaypointCoords, in.TrailSections, in.IsPublished,
		in.Features, in.Activities, in.ElevationGainM, in.RouteType, in.PathPhases,
	)
	if err != nil {
		response.Internal(w, err); return
	}
	response.OK(w, map[string]string{"updated": id})
}

func (s *Service) AdminGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var t AdminTrek
	err := s.pool.QueryRow(r.Context(), `
		SELECT id::text, slug, name, difficulty, trek_type, duration_days, distance_km,
		       max_altitude_m, start_point, end_point, best_months, permits,
		       ams_risk, status, closure_reason, tagline, uniqueness,
		       rating, review_count, guide_available, guide_price_inr,
		       COALESCE(waypoints, '[]'::jsonb), COALESCE(gear_list, '[]'::jsonb),
		       COALESCE(path_geojson, '[]'::jsonb), COALESCE(waypoint_coords, '[]'::jsonb),
		       COALESCE(trail_sections, '[]'::jsonb), is_published,
		       COALESCE(features, '{}'::TEXT[]),
		       COALESCE(activities, '{hike}'::TEXT[]),
		       elevation_gain_m, route_type,
		       COALESCE(path_phases, '[]'::jsonb)
		FROM treks WHERE id = $1
	`, id).Scan(
		&t.ID, &t.Slug, &t.Name, &t.Difficulty, &t.TrekType, &t.DurationDays, &t.DistanceKm,
		&t.MaxAltitudeM, &t.StartPoint, &t.EndPoint, &t.BestMonths, &t.Permits,
		&t.AmsRisk, &t.Status, &t.ClosureReason, &t.Tagline, &t.Uniqueness,
		&t.Rating, &t.ReviewCount, &t.GuideAvailable, &t.GuidePriceINR,
		&t.Waypoints, &t.GearList, &t.PathGeoJSON, &t.WaypointCoords, &t.TrailSections, &t.IsPublished,
		&t.Features, &t.Activities, &t.ElevationGainM, &t.RouteType, &t.PathPhases,
	)
	if err != nil {
		response.Internal(w, err); return
	}
	response.OK(w, t)
}

// GET /admin/treks — list all treks including unpublished
func (s *Service) AdminList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, slug, name, difficulty, trek_type, duration_days, distance_km,
		       max_altitude_m, start_point, end_point, best_months, permits,
		       ams_risk, status, closure_reason, tagline, uniqueness,
		       rating, review_count, guide_available, guide_price_inr, is_published
		FROM treks ORDER BY name
	`)
	if err != nil { response.Internal(w, err); return }
	defer rows.Close()

	out := make([]AdminTrek, 0)
	for rows.Next() {
		var t AdminTrek
		_ = rows.Scan(
			&t.ID, &t.Slug, &t.Name, &t.Difficulty, &t.TrekType, &t.DurationDays, &t.DistanceKm,
			&t.MaxAltitudeM, &t.StartPoint, &t.EndPoint, &t.BestMonths, &t.Permits,
			&t.AmsRisk, &t.Status, &t.ClosureReason, &t.Tagline, &t.Uniqueness,
			&t.Rating, &t.ReviewCount, &t.GuideAvailable, &t.GuidePriceINR, &t.IsPublished,
		)
		out = append(out, t)
	}
	response.OK(w, out)
}

func (s *Service) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.pool.Exec(r.Context(), `DELETE FROM treks WHERE id = $1`, id)
	if err != nil {
		response.Internal(w, err); return
	}
	response.NoContent(w)
}
