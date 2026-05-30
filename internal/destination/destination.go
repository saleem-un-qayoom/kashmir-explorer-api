// Package destination — destinations CRUD + search + nearby + bbox.
package destination

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

type Destination struct {
	ID                   string   `json:"id"`
	Slug                 string   `json:"slug"`
	Name                 string   `json:"name"`
	NameUrdu             *string  `json:"name_urdu,omitempty"`
	NameHindi            *string  `json:"name_hindi,omitempty"`
	District             *string  `json:"district,omitempty"`
	Tagline              *string  `json:"tagline,omitempty"`
	Uniqueness           *string  `json:"uniqueness,omitempty"`
	Lat                  float64  `json:"lat"`
	Lng                  float64  `json:"lng"`
	AltitudeM            *int     `json:"altitude_m,omitempty"`
	BestMonths           []int    `json:"best_months,omitempty"`
	SeasonType           *string  `json:"season_type,omitempty"`
	Rating               float64  `json:"rating"`
	ReviewCount          int      `json:"review_count"`
	DistanceFromSrinagar *int     `json:"distance_from_srinagar_km,omitempty"`
	EntryFeeINR          int      `json:"entry_fee_inr"`
	Permits              []string `json:"permits,omitempty"`
	Categories           []string `json:"categories,omitempty"`
	Features             []string `json:"features,omitempty"`
	Description          *string  `json:"description,omitempty"`
	HeroImageURL         *string  `json:"hero_image_url,omitempty"`
}

// GET /v1/destinations  ?region=&category=&season=&sort=&page=&limit=
func (s *Service) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	region := q.Get("region")
	category := q.Get("category")
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 24
	}
	offset, _ := strconv.Atoi(q.Get("offset"))

	rows, err := s.pool.Query(r.Context(), `
		SELECT d.id::text, d.slug, d.name, d.name_urdu, d.name_hindi, d.district,
		       d.tagline, d.uniqueness,
		       ST_X(d.location::geometry), ST_Y(d.location::geometry),
		       d.altitude_m, d.best_months, d.season_type,
		       d.rating, d.review_count, d.distance_from_srinagar_km, d.entry_fee_inr, d.permits,
		       COALESCE(array_agg(c.slug) FILTER (WHERE c.slug IS NOT NULL), '{}'),
		       COALESCE(d.features, '{}'::TEXT[]),
		       (SELECT url FROM images i
		         WHERE i.destination_id = d.id
		         ORDER BY i.is_hero DESC, i.sort_order, i.created_at LIMIT 1)
		FROM destinations d
		LEFT JOIN regions r ON r.id = d.region_id
		LEFT JOIN destination_categories dc ON dc.destination_id = d.id
		LEFT JOIN categories c ON c.id = dc.category_id
		WHERE d.is_published = true
		  AND ($1 = '' OR r.slug = $1)
		  AND ($2 = '' OR EXISTS (
		    SELECT 1 FROM destination_categories dc2
		    JOIN categories c2 ON c2.id = dc2.category_id
		    WHERE dc2.destination_id = d.id AND c2.slug = $2
		  ))
		GROUP BY d.id
		ORDER BY d.is_featured DESC, d.rating DESC
		LIMIT $3 OFFSET $4
	`, region, category, limit, offset)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()

	out := make([]Destination, 0)
	for rows.Next() {
		var d Destination
		if err := rows.Scan(
			&d.ID, &d.Slug, &d.Name, &d.NameUrdu, &d.NameHindi, &d.District,
			&d.Tagline, &d.Uniqueness, &d.Lng, &d.Lat,
			&d.AltitudeM, &d.BestMonths, &d.SeasonType,
			&d.Rating, &d.ReviewCount, &d.DistanceFromSrinagar, &d.EntryFeeINR, &d.Permits,
			&d.Categories, &d.Features, &d.HeroImageURL,
		); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, d)
	}
	response.OK(w, out)
}

// GET /v1/destinations/featured
func (s *Service) Featured(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, slug, name, tagline, uniqueness, altitude_m, rating
		FROM destinations WHERE is_featured = true AND is_published = true
		ORDER BY rating DESC LIMIT 5
	`)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, slug, name string
		var tagline, uniq *string
		var alt *int
		var rating float64
		if err := rows.Scan(&id, &slug, &name, &tagline, &uniq, &alt, &rating); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, map[string]any{"id": id, "slug": slug, "name": name, "tagline": tagline, "uniqueness": uniq, "altitude_m": alt, "rating": rating})
	}
	response.OK(w, out)
}

// GET /v1/destinations/trending — top-rated published destinations.
// When save_count / view_count data is available this can be re-ranked;
// for now rating DESC is a good proxy.
func (s *Service) Trending(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT d.id::text, d.slug, d.name, d.tagline, d.uniqueness,
		       d.altitude_m, d.rating, d.district,
		       d.distance_from_srinagar_km,
		       (SELECT url FROM images i
		         WHERE i.destination_id = d.id
		         ORDER BY i.is_hero DESC, i.sort_order, i.created_at LIMIT 1)
		FROM destinations d
		WHERE d.is_published = true
		ORDER BY d.is_featured DESC, d.rating DESC
		LIMIT 10
	`)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, slug, name string
		var tagline, uniq, district *string
		var alt, distSgr *int
		var rating float64
		var heroURL *string
		if err := rows.Scan(&id, &slug, &name, &tagline, &uniq, &alt, &rating, &district, &distSgr, &heroURL); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, map[string]any{
			"id": id, "slug": slug, "name": name, "tagline": tagline,
			"uniqueness": uniq, "altitude_m": alt, "rating": rating,
			"district": district, "distance_from_srinagar_km": distSgr,
			"hero_image_url": heroURL,
		})
	}
	response.OK(w, out)
}

// GET /v1/destinations/nearby?lat=&lng=&radius_km=&limit=
func (s *Service) Nearby(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	lat, _ := strconv.ParseFloat(q.Get("lat"), 64)
	lng, _ := strconv.ParseFloat(q.Get("lng"), 64)
	radius, _ := strconv.ParseFloat(q.Get("radius_km"), 64)
	if radius == 0 {
		radius = 20
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit == 0 {
		limit = 10
	}

	rows, err := s.pool.Query(r.Context(), `
		SELECT id::text, slug, name, district, altitude_m, rating,
		       ROUND(ST_Distance(location, ST_GeogFromText('POINT(' || $1 || ' ' || $2 || ')'))::numeric / 1000, 1) AS km
		FROM destinations
		WHERE is_published = true
		  AND ST_DWithin(location, ST_GeogFromText('POINT(' || $1 || ' ' || $2 || ')'), $3 * 1000)
		ORDER BY location <-> ST_GeogFromText('POINT(' || $1 || ' ' || $2 || ')')
		LIMIT $4
	`, lng, lat, radius, limit)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, slug, name, district string
		var alt *int
		var rating, km float64
		if err := rows.Scan(&id, &slug, &name, &district, &alt, &rating, &km); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, map[string]any{"id": id, "slug": slug, "name": name, "district": district, "altitude_m": alt, "rating": rating, "distance_km": km})
	}
	response.OK(w, out)
}

// GET /v1/destinations/map?bbox=minLat,minLng,maxLat,maxLng
func (s *Service) Bbox(w http.ResponseWriter, r *http.Request) {
	bbox := r.URL.Query().Get("bbox")
	parts := splitFloats(bbox)
	if len(parts) != 4 {
		response.BadRequest(w, "bbox must be minLat,minLng,maxLat,maxLng")
		return
	}
	minLat, minLng, maxLat, maxLng := parts[0], parts[1], parts[2], parts[3]
	rows, err := s.pool.Query(r.Context(), `
		SELECT d.id::text, d.slug, d.name,
		       ST_X(d.location::geometry), ST_Y(d.location::geometry),
		       COALESCE(array_agg(c.slug) FILTER (WHERE c.slug IS NOT NULL), '{}'),
		       (SELECT url FROM images i
		         WHERE i.destination_id = d.id
		         ORDER BY i.is_hero DESC, i.sort_order, i.created_at LIMIT 1)
		FROM destinations d
		LEFT JOIN destination_categories dc ON dc.destination_id = d.id
		LEFT JOIN categories c ON c.id = dc.category_id
		WHERE d.is_published = true
		  AND ST_Within(d.location::geometry,
		    ST_MakeEnvelope($1, $2, $3, $4, 4326))
		GROUP BY d.id
		LIMIT 500
	`, minLng, minLat, maxLng, maxLat)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, slug, name string
		var lng, lat float64
		var categories []string
		var heroURL *string
		if err := rows.Scan(&id, &slug, &name, &lng, &lat, &categories, &heroURL); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, map[string]any{
			"id": id, "slug": slug, "name": name, "lng": lng, "lat": lat,
			"categories": categories, "hero_image_url": heroURL,
		})
	}
	response.OK(w, out)
}

// GET /v1/destinations/{slug}
func (s *Service) Get(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	var d Destination
	err := s.pool.QueryRow(r.Context(), `
		SELECT d.id::text, d.slug, d.name, d.name_urdu, d.name_hindi, d.district,
		       d.tagline, d.uniqueness, d.description,
		       ST_X(d.location::geometry), ST_Y(d.location::geometry),
		       d.altitude_m, d.best_months, d.season_type, d.rating, d.review_count,
		       d.distance_from_srinagar_km, d.entry_fee_inr, d.permits,
		       COALESCE(d.features, '{}'::TEXT[]),
		       (SELECT url FROM images i
		         WHERE i.destination_id = d.id
		         ORDER BY i.is_hero DESC, i.sort_order, i.created_at LIMIT 1)
		FROM destinations d WHERE d.slug = $1 AND d.is_published = true
	`, slug).Scan(
		&d.ID, &d.Slug, &d.Name, &d.NameUrdu, &d.NameHindi, &d.District,
		&d.Tagline, &d.Uniqueness, &d.Description,
		&d.Lng, &d.Lat,
		&d.AltitudeM, &d.BestMonths, &d.SeasonType, &d.Rating, &d.ReviewCount,
		&d.DistanceFromSrinagar, &d.EntryFeeINR, &d.Permits,
		&d.Features, &d.HeroImageURL,
	)
	if err != nil {
		response.NotFound(w, "destination not found")
		return
	}
	response.OK(w, d)
}

// GET /v1/categories
func (s *Service) Categories(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `SELECT id::text, name, slug, icon, color FROM categories ORDER BY name`)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, name, slug string
		var icon, color *string
		if err := rows.Scan(&id, &name, &slug, &icon, &color); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, map[string]any{"id": id, "name": name, "slug": slug, "icon": icon, "color": color})
	}
	response.OK(w, out)
}

// GET /v1/regions
func (s *Service) Regions(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `SELECT id::text, name, slug, description FROM regions ORDER BY slug`)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, name, slug string
		var desc *string
		if err := rows.Scan(&id, &name, &slug, &desc); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, map[string]any{"id": id, "name": name, "slug": slug, "description": desc})
	}
	response.OK(w, out)
}

// ─── Admin ────────────────────────────────────────────────────

type AdminDest struct {
	ID              string          `json:"id"`
	Slug            string          `json:"slug"`
	Name            string          `json:"name"`
	NameUrdu        *string         `json:"name_urdu"`
	NameHindi       *string         `json:"name_hindi"`
	District        *string         `json:"district"`
	RegionSlug      *string         `json:"region_slug"`
	Tagline         *string         `json:"tagline"`
	Uniqueness      *string         `json:"uniqueness"`
	Description     *string         `json:"description"`
	Lat             float64         `json:"lat"`
	Lng             float64         `json:"lng"`
	AltitudeM       *int            `json:"altitude_m"`
	BestMonths      []int           `json:"best_months"`
	SeasonType      *string         `json:"season_type"`
	Rating          float64         `json:"rating"`
	ReviewCount     int             `json:"review_count"`
	DistFromSgr     *int            `json:"distance_from_srinagar_km"`
	EntryFee        int             `json:"entry_fee_inr"`
	Permits         []string        `json:"permits"`
	Activities      []string        `json:"activities"`
	NetworkCoverage json.RawMessage `json:"network_coverage"`
	Practical       json.RawMessage `json:"practical"`
	Categories      []string        `json:"categories"`
	IsPublished     bool            `json:"is_published"`
	IsFeatured      bool            `json:"is_featured"`
	IsDeleted       bool            `json:"is_deleted"`
	Features        []string        `json:"features"` // AllTrails-style tags
}

// GET /admin/destinations — all destinations including unpublished
func (s *Service) AdminList(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	where := ""
	switch status {
	case "published":
		where = "WHERE d.is_published = true AND d.is_deleted = false"
	case "unpublished":
		where = "WHERE d.is_published = false AND d.is_deleted = false"
	case "deleted":
		where = "WHERE d.is_deleted = true"
	default:
		where = ""
	}

	query := fmt.Sprintf(`
		SELECT d.id::text, d.slug, d.name, d.name_urdu, d.name_hindi, d.district,
		       r.slug, d.tagline, d.uniqueness, d.description,
		       ST_X(d.location::geometry), ST_Y(d.location::geometry),
		       d.altitude_m, d.best_months, d.season_type,
		       d.rating, d.review_count, d.distance_from_srinagar_km, d.entry_fee_inr,
		       d.permits, d.is_published, d.is_featured, d.is_deleted,
		       d.network_coverage, d.practical,
		       COALESCE(array_agg(DISTINCT c.slug) FILTER (WHERE c.slug IS NOT NULL), '{}'),
		       COALESCE(array_agg(DISTINCT a.activity) FILTER (WHERE a.activity IS NOT NULL), '{}'),
		       COALESCE(d.features, '{}'::TEXT[])
		FROM destinations d
		LEFT JOIN regions r ON r.id = d.region_id
		LEFT JOIN destination_categories dc ON dc.destination_id = d.id
		LEFT JOIN categories c ON c.id = dc.category_id
		LEFT JOIN destination_activities a ON a.destination_id = d.id
		%s
		GROUP BY d.id, r.slug
		ORDER BY d.name
	`, where)

	rows, err := s.pool.Query(r.Context(), query)
	if err != nil {
		response.Internal(w, err)
		return
	}
	defer rows.Close()

	out := make([]AdminDest, 0)
	for rows.Next() {
		var d AdminDest
		if err := rows.Scan(
			&d.ID, &d.Slug, &d.Name, &d.NameUrdu, &d.NameHindi, &d.District,
			&d.RegionSlug, &d.Tagline, &d.Uniqueness, &d.Description,
			&d.Lng, &d.Lat,
			&d.AltitudeM, &d.BestMonths, &d.SeasonType,
			&d.Rating, &d.ReviewCount, &d.DistFromSgr, &d.EntryFee,
			&d.Permits, &d.IsPublished, &d.IsFeatured, &d.IsDeleted,
			&d.NetworkCoverage, &d.Practical,
			&d.Categories, &d.Activities, &d.Features,
		); err != nil {
			response.Internal(w, err)
			return
		}
		out = append(out, d)
	}
	response.OK(w, out)
}

// GET /admin/destinations/{id}
func (s *Service) AdminGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var d AdminDest
	err := s.pool.QueryRow(r.Context(), `
		SELECT d.id::text, d.slug, d.name, d.name_urdu, d.name_hindi, d.district,
		       r.slug, d.tagline, d.uniqueness, d.description,
		       ST_X(d.location::geometry), ST_Y(d.location::geometry),
		       d.altitude_m, d.best_months, d.season_type,
		       d.rating, d.review_count, d.distance_from_srinagar_km, d.entry_fee_inr,
		       d.permits, d.is_published, d.is_featured, d.is_deleted,
		       d.network_coverage, d.practical,
		       COALESCE(array_agg(DISTINCT c.slug) FILTER (WHERE c.slug IS NOT NULL), '{}'),
		       COALESCE(array_agg(DISTINCT a.activity) FILTER (WHERE a.activity IS NOT NULL), '{}'),
		       COALESCE(d.features, '{}'::TEXT[])
		FROM destinations d
		LEFT JOIN regions r ON r.id = d.region_id
		LEFT JOIN destination_categories dc ON dc.destination_id = d.id
		LEFT JOIN categories c ON c.id = dc.category_id
		LEFT JOIN destination_activities a ON a.destination_id = d.id
		WHERE d.id = $1
		GROUP BY d.id, r.slug
	`, id).Scan(
		&d.ID, &d.Slug, &d.Name, &d.NameUrdu, &d.NameHindi, &d.District,
		&d.RegionSlug, &d.Tagline, &d.Uniqueness, &d.Description,
		&d.Lng, &d.Lat,
		&d.AltitudeM, &d.BestMonths, &d.SeasonType,
		&d.Rating, &d.ReviewCount, &d.DistFromSgr, &d.EntryFee,
		&d.Permits, &d.IsPublished, &d.IsFeatured, &d.IsDeleted,
		&d.NetworkCoverage, &d.Practical,
		&d.Categories, &d.Activities, &d.Features,
	)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, d)
}

type AdminDestInput struct {
	Name            string          `json:"name"`
	NameUrdu        *string         `json:"name_urdu"`
	NameHindi       *string         `json:"name_hindi"`
	Slug            string          `json:"slug"`
	RegionSlug      string          `json:"region_slug"`
	District        *string         `json:"district"`
	Tagline         *string         `json:"tagline"`
	Uniqueness      *string         `json:"uniqueness"`
	Lat             float64         `json:"lat"`
	Lng             float64         `json:"lng"`
	AltitudeM       *int            `json:"altitude_m"`
	BestMonths      []int           `json:"best_months"`
	SeasonType      *string         `json:"season_type"`
	DistFromSgr     *int            `json:"distance_from_srinagar_km"`
	EntryFee        int             `json:"entry_fee_inr"`
	Permits         []string        `json:"permits"`
	Activities      []string        `json:"activities"`
	NetworkCoverage json.RawMessage `json:"network_coverage"`
	Practical       json.RawMessage `json:"practical"`
	Categories      []string        `json:"categories"`
	IsPublished     bool            `json:"is_published"`
	IsFeatured      bool            `json:"is_featured"`
	Description     *string         `json:"description"`
	Features        []string        `json:"features"` // AllTrails-style tags (migration 0010)
}

func (s *Service) AdminCreate(w http.ResponseWriter, r *http.Request) {
	var in AdminDestInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	if in.Name == "" || in.Slug == "" {
		response.BadRequest(w, "name and slug required")
		return
	}

	var id string
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO destinations
			(name, name_urdu, name_hindi, slug, region_id, district, tagline, uniqueness,
			 description, location, altitude_m, best_months, season_type,
			 distance_from_srinagar_km, entry_fee_inr, permits,
			 network_coverage, practical,
			 is_published, is_featured, features)
		VALUES ($1, $2, $3, $4,
			(SELECT id FROM regions WHERE slug = $5),
			$6, $7, $8, $9,
			CASE WHEN $10::float8 IS NOT NULL AND $11::float8 IS NOT NULL
			     THEN ST_SetSRID(ST_MakePoint($10, $11), 4326)::geography
			     ELSE NULL END,
			$12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
		RETURNING id::text
	`, in.Name, in.NameUrdu, in.NameHindi, in.Slug, in.RegionSlug,
		in.District, in.Tagline, in.Uniqueness, in.Description,
		in.Lng, in.Lat, in.AltitudeM, in.BestMonths, in.SeasonType,
		in.DistFromSgr, in.EntryFee, in.Permits,
		in.NetworkCoverage, in.Practical,
		in.IsPublished, in.IsFeatured, in.Features,
	).Scan(&id)
	if err != nil {
		response.Internal(w, err)
		return
	}

	// Link categories
	for _, slug := range in.Categories {
		_, _ = s.pool.Exec(r.Context(), `
			INSERT INTO destination_categories (destination_id, category_id)
			SELECT $1::uuid, id FROM categories WHERE slug = $2
			ON CONFLICT DO NOTHING
		`, id, slug)
	}
	// Link activities
	for _, act := range in.Activities {
		_, _ = s.pool.Exec(r.Context(), `
			INSERT INTO destination_activities (destination_id, activity)
			VALUES ($1, $2) ON CONFLICT DO NOTHING
		`, id, act)
	}

	response.OK(w, map[string]string{"id": id})
}

func (s *Service) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in AdminDestInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}

	_, err := s.pool.Exec(r.Context(), `
		UPDATE destinations SET
			name = $1, name_urdu = $2, name_hindi = $3, slug = $4,
			region_id = (SELECT id FROM regions WHERE slug = $5),
			district = $6, tagline = $7, uniqueness = $8, description = $9,
			location = CASE WHEN $10::float8 IS NOT NULL AND $11::float8 IS NOT NULL
			                THEN ST_SetSRID(ST_MakePoint($10, $11), 4326)::geography
			                ELSE location END,
			altitude_m = $12, best_months = $13, season_type = $14,
			distance_from_srinagar_km = $15, entry_fee_inr = $16,
			permits = $17,
			network_coverage = $18, practical = $19,
			is_published = $20, is_featured = $21,
			features = $22,
			updated_at = now()
		WHERE id = $23
	`, in.Name, in.NameUrdu, in.NameHindi, in.Slug, in.RegionSlug,
		in.District, in.Tagline, in.Uniqueness, in.Description,
		in.Lng, in.Lat, in.AltitudeM, in.BestMonths, in.SeasonType,
		in.DistFromSgr, in.EntryFee, in.Permits,
		in.NetworkCoverage, in.Practical,
		in.IsPublished, in.IsFeatured, in.Features,
		id,
	)
	if err != nil {
		response.Internal(w, err)
		return
	}

	// Re-link categories
	_, _ = s.pool.Exec(r.Context(), `DELETE FROM destination_categories WHERE destination_id = $1`, id)
	for _, slug := range in.Categories {
		_, _ = s.pool.Exec(r.Context(), `
			INSERT INTO destination_categories (destination_id, category_id)
			SELECT $1::uuid, id FROM categories WHERE slug = $2
			ON CONFLICT DO NOTHING
		`, id, slug)
	}
	// Re-link activities
	_, _ = s.pool.Exec(r.Context(), `DELETE FROM destination_activities WHERE destination_id = $1`, id)
	for _, act := range in.Activities {
		_, _ = s.pool.Exec(r.Context(), `
			INSERT INTO destination_activities (destination_id, activity)
			VALUES ($1, $2) ON CONFLICT DO NOTHING
		`, id, act)
	}

	response.OK(w, map[string]string{"updated": id})
}

func (s *Service) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.pool.Exec(r.Context(), `UPDATE destinations SET is_deleted = true, is_published = false WHERE id = $1`, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, map[string]string{"deleted": id})
}

// POST /admin/destinations/{id}/restore
func (s *Service) AdminRestore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.pool.Exec(r.Context(), `UPDATE destinations SET is_deleted = false WHERE id = $1`, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, map[string]string{"restored": id})
}

// DELETE /admin/destinations/{id}/permanent
func (s *Service) AdminDeletePermanent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.pool.Exec(r.Context(), `DELETE FROM destinations WHERE id = $1`, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, map[string]string{"deleted": id})
}

// ─── Admin: Categories ──────────────────────────────────────────

func (s *Service) AdminCategoryGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var c struct {
		ID    string  `json:"id"`
		Name  string  `json:"name"`
		Slug  string  `json:"slug"`
		Icon  *string `json:"icon"`
		Color *string `json:"color"`
	}
	err := s.pool.QueryRow(r.Context(), `
		SELECT id::text, name, slug, icon, color FROM categories WHERE id = $1
	`, id).Scan(&c.ID, &c.Name, &c.Slug, &c.Icon, &c.Color)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, c)
}

func (s *Service) AdminCategoryCreate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name  string  `json:"name"`
		Slug  string  `json:"slug"`
		Icon  *string `json:"icon"`
		Color *string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	var id string
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO categories (name, slug, icon, color)
		VALUES ($1, $2, $3, $4) RETURNING id::text
	`, in.Name, in.Slug, in.Icon, in.Color).Scan(&id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.Created(w, map[string]string{"id": id})
}

func (s *Service) AdminCategoryUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in struct {
		Name  string  `json:"name"`
		Slug  string  `json:"slug"`
		Icon  *string `json:"icon"`
		Color *string `json:"color"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	_, err := s.pool.Exec(r.Context(), `
		UPDATE categories SET name = $1, slug = $2, icon = $3, color = $4 WHERE id = $5
	`, in.Name, in.Slug, in.Icon, in.Color, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, map[string]string{"updated": id})
}

func (s *Service) AdminCategoryDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.pool.Exec(r.Context(), `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.NoContent(w)
}

// ─── Admin: Regions ─────────────────────────────────────────────

func (s *Service) AdminRegionGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var reg struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		Description *string `json:"description"`
	}
	err := s.pool.QueryRow(r.Context(), `
		SELECT id::text, name, slug, description FROM regions WHERE id = $1
	`, id).Scan(&reg.ID, &reg.Name, &reg.Slug, &reg.Description)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, reg)
}

func (s *Service) AdminRegionCreate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	var id string
	err := s.pool.QueryRow(r.Context(), `
		INSERT INTO regions (name, slug, description) VALUES ($1, $2, $3) RETURNING id::text
	`, in.Name, in.Slug, in.Description).Scan(&id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.Created(w, map[string]string{"id": id})
}

func (s *Service) AdminRegionUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in struct {
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	_, err := s.pool.Exec(r.Context(), `
		UPDATE regions SET name = $1, slug = $2, description = $3 WHERE id = $4
	`, in.Name, in.Slug, in.Description, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, map[string]string{"updated": id})
}

func (s *Service) AdminRegionDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.pool.Exec(r.Context(), `DELETE FROM regions WHERE id = $1`, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.NoContent(w)
}

// ─── helpers ────────────────────────────────────────────────────

func splitFloats(s string) []float64 {
	out := []float64{}
	current := ""
	for _, ch := range s {
		if ch == ',' {
			if v, err := strconv.ParseFloat(current, 64); err == nil {
				out = append(out, v)
			}
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		if v, err := strconv.ParseFloat(current, 64); err == nil {
			out = append(out, v)
		}
	}
	return out
}
