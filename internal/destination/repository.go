package destination

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository owns all database access for the destination domain. SQL is moved
// verbatim from the pre-refactor handlers; only the scan targets changed from
// map[string]any to the typed structs in dto.go.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// ─── Public reads ───────────────────────────────────────────────

func (r *Repository) List(ctx context.Context, region, category string, limit, offset int) ([]Destination, error) {
	rows, err := r.pool.Query(ctx, `
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
		return nil, err
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
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) Featured(ctx context.Context) ([]FeaturedDestination, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, slug, name, tagline, uniqueness, altitude_m, rating
		FROM destinations WHERE is_featured = true AND is_published = true
		ORDER BY rating DESC LIMIT 5
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []FeaturedDestination{}
	for rows.Next() {
		var d FeaturedDestination
		if err := rows.Scan(&d.ID, &d.Slug, &d.Name, &d.Tagline, &d.Uniqueness, &d.AltitudeM, &d.Rating); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) Trending(ctx context.Context) ([]TrendingDestination, error) {
	rows, err := r.pool.Query(ctx, `
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
		return nil, err
	}
	defer rows.Close()

	out := []TrendingDestination{}
	for rows.Next() {
		var d TrendingDestination
		if err := rows.Scan(&d.ID, &d.Slug, &d.Name, &d.Tagline, &d.Uniqueness,
			&d.AltitudeM, &d.Rating, &d.District, &d.DistanceFromSrinagar, &d.HeroImageURL); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) Nearby(ctx context.Context, lng, lat, radius float64, limit int) ([]NearbyDestination, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, slug, name, district, altitude_m, rating,
		       ROUND(ST_Distance(location, ST_GeogFromText('POINT(' || $1 || ' ' || $2 || ')'))::numeric / 1000, 1) AS km
		FROM destinations
		WHERE is_published = true
		  AND ST_DWithin(location, ST_GeogFromText('POINT(' || $1 || ' ' || $2 || ')'), $3 * 1000)
		ORDER BY location <-> ST_GeogFromText('POINT(' || $1 || ' ' || $2 || ')')
		LIMIT $4
	`, lng, lat, radius, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []NearbyDestination{}
	for rows.Next() {
		var d NearbyDestination
		if err := rows.Scan(&d.ID, &d.Slug, &d.Name, &d.District, &d.AltitudeM, &d.Rating, &d.DistanceKm); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) Bbox(ctx context.Context, minLng, minLat, maxLng, maxLat float64) ([]MapPin, error) {
	rows, err := r.pool.Query(ctx, `
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
		return nil, err
	}
	defer rows.Close()

	out := []MapPin{}
	for rows.Next() {
		var p MapPin
		if err := rows.Scan(&p.ID, &p.Slug, &p.Name, &p.Lng, &p.Lat, &p.Categories, &p.HeroImageURL); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// GetBySlug returns the public detail for one destination. The error (incl.
// pgx.ErrNoRows) is surfaced to the caller, which maps it to 404.
func (r *Repository) GetBySlug(ctx context.Context, slug string) (*Destination, error) {
	var d Destination
	err := r.pool.QueryRow(ctx, `
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
		return nil, err
	}
	return &d, nil
}

func (r *Repository) Categories(ctx context.Context) ([]Category, error) {
	rows, err := r.pool.Query(ctx, `SELECT id::text, name, slug, icon, color FROM categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Icon, &c.Color); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) Regions(ctx context.Context) ([]Region, error) {
	rows, err := r.pool.Query(ctx, `SELECT id::text, name, slug, description FROM regions ORDER BY slug`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Region{}
	for rows.Next() {
		var reg Region
		if err := rows.Scan(&reg.ID, &reg.Name, &reg.Slug, &reg.Description); err != nil {
			return nil, err
		}
		out = append(out, reg)
	}
	return out, rows.Err()
}

// ─── Admin: destinations ────────────────────────────────────────

func (r *Repository) AdminList(ctx context.Context, status string) ([]AdminDest, error) {
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

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) AdminGet(ctx context.Context, id string) (*AdminDest, error) {
	var d AdminDest
	err := r.pool.QueryRow(ctx, `
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
		return nil, err
	}
	return &d, nil
}

// AdminCreate inserts a destination and links its categories/activities,
// returning the new id. The link inserts intentionally ignore errors to match
// the prior behavior.
func (r *Repository) AdminCreate(ctx context.Context, in AdminDestInput) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
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
		return "", err
	}

	for _, slug := range in.Categories {
		_, _ = r.pool.Exec(ctx, `
			INSERT INTO destination_categories (destination_id, category_id)
			SELECT $1::uuid, id FROM categories WHERE slug = $2
			ON CONFLICT DO NOTHING
		`, id, slug)
	}
	for _, act := range in.Activities {
		_, _ = r.pool.Exec(ctx, `
			INSERT INTO destination_activities (destination_id, activity)
			VALUES ($1, $2) ON CONFLICT DO NOTHING
		`, id, act)
	}
	return id, nil
}

// AdminUpdate updates a destination and re-links its categories/activities.
func (r *Repository) AdminUpdate(ctx context.Context, id string, in AdminDestInput) error {
	_, err := r.pool.Exec(ctx, `
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
		return err
	}

	_, _ = r.pool.Exec(ctx, `DELETE FROM destination_categories WHERE destination_id = $1`, id)
	for _, slug := range in.Categories {
		_, _ = r.pool.Exec(ctx, `
			INSERT INTO destination_categories (destination_id, category_id)
			SELECT $1::uuid, id FROM categories WHERE slug = $2
			ON CONFLICT DO NOTHING
		`, id, slug)
	}
	_, _ = r.pool.Exec(ctx, `DELETE FROM destination_activities WHERE destination_id = $1`, id)
	for _, act := range in.Activities {
		_, _ = r.pool.Exec(ctx, `
			INSERT INTO destination_activities (destination_id, activity)
			VALUES ($1, $2) ON CONFLICT DO NOTHING
		`, id, act)
	}
	return nil
}

func (r *Repository) AdminSoftDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE destinations SET is_deleted = true, is_published = false WHERE id = $1`, id)
	return err
}

func (r *Repository) AdminRestore(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `UPDATE destinations SET is_deleted = false WHERE id = $1`, id)
	return err
}

func (r *Repository) AdminDeletePermanent(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM destinations WHERE id = $1`, id)
	return err
}

// ─── Admin: categories ──────────────────────────────────────────

func (r *Repository) CategoryGet(ctx context.Context, id string) (*Category, error) {
	var c Category
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, name, slug, icon, color FROM categories WHERE id = $1
	`, id).Scan(&c.ID, &c.Name, &c.Slug, &c.Icon, &c.Color)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) CategoryCreate(ctx context.Context, in CategoryInput) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO categories (name, slug, icon, color)
		VALUES ($1, $2, $3, $4) RETURNING id::text
	`, in.Name, in.Slug, in.Icon, in.Color).Scan(&id)
	return id, err
}

func (r *Repository) CategoryUpdate(ctx context.Context, id string, in CategoryInput) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE categories SET name = $1, slug = $2, icon = $3, color = $4 WHERE id = $5
	`, in.Name, in.Slug, in.Icon, in.Color, id)
	return err
}

func (r *Repository) CategoryDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM categories WHERE id = $1`, id)
	return err
}

// ─── Admin: regions ─────────────────────────────────────────────

func (r *Repository) RegionGet(ctx context.Context, id string) (*Region, error) {
	var reg Region
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, name, slug, description FROM regions WHERE id = $1
	`, id).Scan(&reg.ID, &reg.Name, &reg.Slug, &reg.Description)
	if err != nil {
		return nil, err
	}
	return &reg, nil
}

func (r *Repository) RegionCreate(ctx context.Context, in RegionInput) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO regions (name, slug, description) VALUES ($1, $2, $3) RETURNING id::text
	`, in.Name, in.Slug, in.Description).Scan(&id)
	return id, err
}

func (r *Repository) RegionUpdate(ctx context.Context, id string, in RegionInput) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE regions SET name = $1, slug = $2, description = $3 WHERE id = $4
	`, in.Name, in.Slug, in.Description, id)
	return err
}

func (r *Repository) RegionDelete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM regions WHERE id = $1`, id)
	return err
}
