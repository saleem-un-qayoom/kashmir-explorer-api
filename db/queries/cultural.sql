-- name: ListFood :many
SELECT id, name,
       COALESCE(name_local->>'ur', '') AS name_urdu,
       COALESCE(name_local->>'ks', '') AS name_kashmiri,
       COALESCE((details->>'vegetarian')::boolean, false) AS vegetarian,
       COALESCE(description, '') AS description,
       COALESCE(details->>'where_to_try', '') AS where_to_try,
       COALESCE(details->>'price_range', '') AS price_range
FROM cultural_items WHERE type = 'dish'
ORDER BY name;

-- name: ListFestivals :many
SELECT id, name,
       COALESCE((details->>'month')::int, 0) AS month,
       COALESCE(details->>'duration', '') AS duration,
       COALESCE(description, '') AS description,
       COALESCE(details->>'region', '') AS region
FROM cultural_items WHERE type = 'festival'
ORDER BY (details->>'month')::int NULLS LAST;

-- name: ListCrafts :many
SELECT id, name,
       COALESCE(details->>'origin', '') AS origin,
       COALESCE(details->>'price', '') AS price,
       COALESCE(description, '') AS description
FROM cultural_items WHERE type = 'craft' ORDER BY name;

-- name: ListEtiquette :many
SELECT COALESCE(details->>'category', '') AS category,
       name AS title,
       COALESCE(description, '') AS body
FROM cultural_items WHERE type = 'etiquette'
ORDER BY (details->>'category');

-- name: CreateCulturalItem :exec
INSERT INTO cultural_items (type, name, description, details, name_local)
VALUES ($1, $2, $3, $4, $5);

-- name: ListPhotoSpotsForDestination :many
SELECT ps.id, ps.name,
       ST_X(ps.location::geometry) AS lng,
       ST_Y(ps.location::geometry) AS lat,
       COALESCE(ps.best_time, '') AS best_time,
       COALESCE(ps.facing, '') AS facing,
       COALESCE(ps.tripod_recommended, false) AS tripod_recommended,
       COALESCE(ps.drone_allowed, false) AS drone_allowed,
       COALESCE(ps.description, '') AS description
FROM photo_spots ps
JOIN destinations d ON d.id = ps.destination_id
WHERE d.slug = $1
ORDER BY ps.name;

-- name: ListPermits :many
SELECT id, name, required, office, processing_days, cost_inr, validity,
       status, notes, official_url
FROM permits ORDER BY id;

-- name: PermitsForDestinations :many
SELECT DISTINCT unnest(d.permits) AS permit_key
FROM destinations d
WHERE d.slug = ANY($1::text[])
  AND COALESCE(array_length(d.permits, 1), 0) > 0;
