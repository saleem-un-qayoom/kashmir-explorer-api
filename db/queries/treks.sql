-- name: ListTreks :many
SELECT id, slug, name, difficulty, trek_type, duration_days, distance_km,
       max_altitude_m, start_point, end_point, best_months, permits,
       ams_risk, status, closure_reason, tagline, uniqueness,
       rating, review_count, guide_available, guide_price_inr
FROM treks
WHERE is_published = true
  AND (sqlc.narg('difficulty')::text IS NULL OR difficulty = sqlc.narg('difficulty')::text)
  AND (NOT sqlc.arg('open_only')::bool OR status = 'open')
ORDER BY rating DESC
LIMIT sqlc.arg('lim')::int;

-- name: GetTrekBySlug :one
SELECT id, slug, name, destination_id, difficulty, trek_type,
       duration_days, distance_km, max_altitude_m,
       start_point, end_point, best_months, permits,
       ams_risk, status, closure_reason, tagline, uniqueness,
       rating, review_count, guide_available, guide_price_inr,
       waypoints, gear_list
FROM treks
WHERE slug = $1 AND is_published = true;

-- name: GetTrekPath :one
SELECT
  COALESCE(path_geojson::text, '[]') AS polyline,
  COALESCE(waypoint_coords::text, '[]') AS waypoints,
  COALESCE(distance_km, 0) AS distance_km
FROM treks
WHERE slug = $1 AND is_published = true;
