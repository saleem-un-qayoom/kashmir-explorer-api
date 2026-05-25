-- name: ListDestinations :many
-- List published destinations with optional region/category filters.
SELECT d.id, d.slug, d.name, d.name_urdu, d.name_hindi, d.district,
       d.tagline, d.uniqueness,
       ST_X(d.location::geometry) AS lng,
       ST_Y(d.location::geometry) AS lat,
       d.altitude_m, d.best_months, d.season_type,
       d.rating, d.review_count, d.distance_from_srinagar_km, d.entry_fee_inr, d.permits
FROM destinations d
LEFT JOIN regions r ON r.id = d.region_id
LEFT JOIN destination_categories dc ON dc.destination_id = d.id
LEFT JOIN categories c ON c.id = dc.category_id
WHERE d.is_published = true
  AND (sqlc.narg('region')::text IS NULL OR r.slug = sqlc.narg('region')::text)
  AND (sqlc.narg('category')::text IS NULL OR EXISTS (
    SELECT 1 FROM destination_categories dc2
    JOIN categories c2 ON c2.id = dc2.category_id
    WHERE dc2.destination_id = d.id AND c2.slug = sqlc.narg('category')::text
  ))
GROUP BY d.id
ORDER BY d.is_featured DESC, d.rating DESC
LIMIT sqlc.arg('lim')::int OFFSET sqlc.arg('off')::int;

-- name: GetDestinationBySlug :one
SELECT id, slug, name, name_urdu, name_hindi, district, tagline, uniqueness,
       ST_X(location::geometry) AS lng,
       ST_Y(location::geometry) AS lat,
       altitude_m, best_months, season_type, rating, review_count,
       distance_from_srinagar_km, entry_fee_inr, permits,
       network_coverage, practical
FROM destinations
WHERE slug = $1 AND is_published = true;

-- name: NearbyDestinations :many
SELECT id, slug, name, district, altitude_m, rating,
       ROUND(ST_Distance(
         location,
         ST_GeogFromText('POINT(' || sqlc.arg('lng')::float || ' ' || sqlc.arg('lat')::float || ')')
       )::numeric / 1000, 1) AS distance_km
FROM destinations
WHERE is_published = true
  AND ST_DWithin(
    location,
    ST_GeogFromText('POINT(' || sqlc.arg('lng')::float || ' ' || sqlc.arg('lat')::float || ')'),
    sqlc.arg('radius_km')::float * 1000
  )
ORDER BY location <-> ST_GeogFromText('POINT(' || sqlc.arg('lng')::float || ' ' || sqlc.arg('lat')::float || ')')
LIMIT sqlc.arg('lim')::int;

-- name: DestinationsInBbox :many
SELECT id, slug, name,
       ST_X(location::geometry) AS lng,
       ST_Y(location::geometry) AS lat
FROM destinations
WHERE is_published = true
  AND ST_Within(
    location::geometry,
    ST_MakeEnvelope(sqlc.arg('min_lng')::float, sqlc.arg('min_lat')::float,
                    sqlc.arg('max_lng')::float, sqlc.arg('max_lat')::float, 4326)
  )
LIMIT 500;

-- name: FeaturedDestinations :many
SELECT id, slug, name, tagline, uniqueness, altitude_m, rating
FROM destinations
WHERE is_featured = true AND is_published = true
ORDER BY rating DESC
LIMIT 5;

-- name: CreateDestination :one
INSERT INTO destinations
  (name, slug, region_id, district, tagline, uniqueness, location,
   altitude_m, best_months, season_type, permits, is_published)
VALUES
  (sqlc.arg('name'), sqlc.arg('slug'), sqlc.arg('region_id'),
   sqlc.arg('district'), sqlc.arg('tagline'), sqlc.arg('uniqueness'),
   ST_GeogFromText('POINT(' || sqlc.arg('lng')::float || ' ' || sqlc.arg('lat')::float || ')'),
   sqlc.arg('altitude_m'), sqlc.arg('best_months')::int[],
   sqlc.arg('season_type'), sqlc.arg('permits')::text[], sqlc.arg('is_published'))
RETURNING id;
