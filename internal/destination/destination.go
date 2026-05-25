// Package destination — destinations CRUD + search + nearby + bbox.
package destination

import (
	"encoding/json"
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
	ID         string  `json:"id"`
	Slug       string  `json:"slug"`
	Name       string  `json:"name"`
	NameUrdu   *string `json:"name_urdu,omitempty"`
	NameHindi  *string `json:"name_hindi,omitempty"`
	District   *string `json:"district,omitempty"`
	Tagline    *string `json:"tagline,omitempty"`
	Uniqueness *string `json:"uniqueness,omitempty"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	AltitudeM  *int    `json:"altitude_m,omitempty"`
	BestMonths []int   `json:"best_months,omitempty"`
	SeasonType *string `json:"season_type,omitempty"`
	Rating     float64 `json:"rating"`
	ReviewCount int    `json:"review_count"`
	DistanceFromSrinagar *int `json:"distance_from_srinagar_km,omitempty"`
	EntryFeeINR int    `json:"entry_fee_inr"`
	Permits    []string `json:"permits,omitempty"`
	Categories []string `json:"categories,omitempty"`
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
		       COALESCE(array_agg(c.slug) FILTER (WHERE c.slug IS NOT NULL), '{}')
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
			&d.Categories,
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
		_ = rows.Scan(&id, &slug, &name, &tagline, &uniq, &alt, &rating)
		out = append(out, map[string]any{"id": id, "slug": slug, "name": name, "tagline": tagline, "uniqueness": uniq, "altitude_m": alt, "rating": rating})
	}
	response.OK(w, out)
}

// GET /v1/destinations/trending — sorted by recent saves (stubbed = rating).
func (s *Service) Trending(w http.ResponseWriter, r *http.Request) {
	s.Featured(w, r) // identical shape for now
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
		_ = rows.Scan(&id, &slug, &name, &district, &alt, &rating, &km)
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
		SELECT id::text, slug, name, ST_X(location::geometry), ST_Y(location::geometry)
		FROM destinations
		WHERE is_published = true
		  AND ST_Within(location::geometry,
		    ST_MakeEnvelope($1, $2, $3, $4, 4326))
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
		_ = rows.Scan(&id, &slug, &name, &lng, &lat)
		out = append(out, map[string]any{"id": id, "slug": slug, "name": name, "lng": lng, "lat": lat})
	}
	response.OK(w, out)
}

// GET /v1/destinations/{slug}
func (s *Service) Get(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	var d Destination
	err := s.pool.QueryRow(r.Context(), `
		SELECT id::text, slug, name, name_urdu, name_hindi, district,
		       tagline, uniqueness,
		       ST_X(location::geometry), ST_Y(location::geometry),
		       altitude_m, best_months, season_type, rating, review_count,
		       distance_from_srinagar_km, entry_fee_inr, permits
		FROM destinations WHERE slug = $1 AND is_published = true
	`, slug).Scan(
		&d.ID, &d.Slug, &d.Name, &d.NameUrdu, &d.NameHindi, &d.District,
		&d.Tagline, &d.Uniqueness, &d.Lng, &d.Lat,
		&d.AltitudeM, &d.BestMonths, &d.SeasonType, &d.Rating, &d.ReviewCount,
		&d.DistanceFromSrinagar, &d.EntryFeeINR, &d.Permits,
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
		_ = rows.Scan(&id, &name, &slug, &icon, &color)
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
		_ = rows.Scan(&id, &name, &slug, &desc)
		out = append(out, map[string]any{"id": id, "name": name, "slug": slug, "description": desc})
	}
	response.OK(w, out)
}

// ─── Admin ────────────────────────────────────────────────────

type AdminDest struct {
	ID          string   `json:"id"`
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	NameUrdu    *string  `json:"name_urdu"`
	NameHindi   *string  `json:"name_hindi"`
	District    *string  `json:"district"`
	RegionSlug  *string  `json:"region_slug"`
	Tagline     *string  `json:"tagline"`
	Uniqueness  *string  `json:"uniqueness"`
	Description *string  `json:"description"`
	Lat         float64  `json:"lat"`
	Lng         float64  `json:"lng"`
	AltitudeM   *int     `json:"altitude_m"`
	BestMonths  []int    `json:"best_months"`
	SeasonType  *string  `json:"season_type"`
	Rating      float64  `json:"rating"`
	ReviewCount int      `json:"review_count"`
	DistFromSgr *int     `json:"distance_from_srinagar_km"`
	EntryFee    int      `json:"entry_fee_inr"`
	Permits     []string `json:"permits"`
	Categories  []string `json:"categories"`
	IsPublished bool     `json:"is_published"`
	IsFeatured  bool     `json:"is_featured"`
}

// GET /admin/destinations — all destinations including unpublished
func (s *Service) AdminList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pool.Query(r.Context(), `
		SELECT d.id::text, d.slug, d.name, d.name_urdu, d.name_hindi, d.district,
		       r.slug, d.tagline, d.uniqueness, d.description,
		       ST_X(d.location::geometry), ST_Y(d.location::geometry),
		       d.altitude_m, d.best_months, d.season_type,
		       d.rating, d.review_count, d.distance_from_srinagar_km, d.entry_fee_inr,
		       d.permits, d.is_published, d.is_featured,
		       COALESCE(array_agg(c.slug) FILTER (WHERE c.slug IS NOT NULL), '{}')
		FROM destinations d
		LEFT JOIN regions r ON r.id = d.region_id
		LEFT JOIN destination_categories dc ON dc.destination_id = d.id
		LEFT JOIN categories c ON c.id = dc.category_id
		GROUP BY d.id, r.slug
		ORDER BY d.name
	`)
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
			&d.Permits, &d.IsPublished, &d.IsFeatured,
			&d.Categories,
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
		       d.permits, d.is_published, d.is_featured,
		       COALESCE(array_agg(c.slug) FILTER (WHERE c.slug IS NOT NULL), '{}')
		FROM destinations d
		LEFT JOIN regions r ON r.id = d.region_id
		LEFT JOIN destination_categories dc ON dc.destination_id = d.id
		LEFT JOIN categories c ON c.id = dc.category_id
		WHERE d.id = $1
		GROUP BY d.id, r.slug
	`, id).Scan(
		&d.ID, &d.Slug, &d.Name, &d.NameUrdu, &d.NameHindi, &d.District,
		&d.RegionSlug, &d.Tagline, &d.Uniqueness, &d.Description,
		&d.Lng, &d.Lat,
		&d.AltitudeM, &d.BestMonths, &d.SeasonType,
		&d.Rating, &d.ReviewCount, &d.DistFromSgr, &d.EntryFee,
		&d.Permits, &d.IsPublished, &d.IsFeatured,
		&d.Categories,
	)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, d)
}

type AdminDestInput struct {
	Name        string   `json:"name"`
	NameUrdu    *string  `json:"name_urdu"`
	NameHindi   *string  `json:"name_hindi"`
	Slug        string   `json:"slug"`
	RegionSlug  string   `json:"region_slug"`
	District    *string  `json:"district"`
	Tagline     *string  `json:"tagline"`
	Uniqueness  *string  `json:"uniqueness"`
	Lat         float64  `json:"lat"`
	Lng         float64  `json:"lng"`
	AltitudeM   *int     `json:"altitude_m"`
	BestMonths  []int    `json:"best_months"`
	SeasonType  *string  `json:"season_type"`
	DistFromSgr *int     `json:"distance_from_srinagar_km"`
	EntryFee    int      `json:"entry_fee_inr"`
	Permits     []string `json:"permits"`
	Categories  []string `json:"categories"`
	IsPublished bool     `json:"is_published"`
	IsFeatured  bool     `json:"is_featured"`
	Description *string  `json:"description"`
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
			 distance_from_srinagar_km, entry_fee_inr, permits, is_published, is_featured)
		VALUES ($1, $2, $3, $4,
			(SELECT id FROM regions WHERE slug = $5),
			$6, $7, $8, $9,
			ST_GeogFromText('POINT(' || $10::text || ' ' || $11::text || ')'),
			$12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id::text
	`, in.Name, in.NameUrdu, in.NameHindi, in.Slug, in.RegionSlug,
		in.District, in.Tagline, in.Uniqueness, in.Description,
		in.Lng, in.Lat, in.AltitudeM, in.BestMonths, in.SeasonType,
		in.DistFromSgr, in.EntryFee, in.Permits, in.IsPublished, in.IsFeatured,
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
			location = ST_GeogFromText('POINT(' || $10::text || ' ' || $11::text || ')'),
			altitude_m = $12, best_months = $13, season_type = $14,
			distance_from_srinagar_km = $15, entry_fee_inr = $16,
			permits = $17, is_published = $18, is_featured = $19,
			updated_at = now()
		WHERE id = $20
	`, in.Name, in.NameUrdu, in.NameHindi, in.Slug, in.RegionSlug,
		in.District, in.Tagline, in.Uniqueness, in.Description,
		in.Lng, in.Lat, in.AltitudeM, in.BestMonths, in.SeasonType,
		in.DistFromSgr, in.EntryFee, in.Permits, in.IsPublished, in.IsFeatured,
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

	response.OK(w, map[string]string{"updated": id})
}

func (s *Service) AdminDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := s.pool.Exec(r.Context(), `UPDATE destinations SET is_published = false WHERE id = $1`, id)
	if err != nil {
		response.Internal(w, err)
		return
	}
	response.OK(w, map[string]string{"unpublished": id})
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
