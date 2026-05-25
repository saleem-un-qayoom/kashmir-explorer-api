-- name: GetUser :one
SELECT id, name, email, phone, role, language, avatar_url
FROM users WHERE id = $1;

-- name: UpsertUserByPhone :one
INSERT INTO users (phone, provider, role)
VALUES ($1, 'phone', 'user')
ON CONFLICT (phone) DO UPDATE SET updated_at = now()
RETURNING id, role;

-- name: UpsertUserByOAuth :one
INSERT INTO users (email, provider, provider_id, name, avatar_url, role)
VALUES ($1, $2, $3, $4, $5, 'user')
ON CONFLICT (email) DO UPDATE
  SET provider = EXCLUDED.provider,
      provider_id = EXCLUDED.provider_id,
      updated_at = now()
RETURNING id, role;

-- name: UpdateUser :exec
UPDATE users SET
  name       = COALESCE(sqlc.narg('name'),     name),
  language   = COALESCE(sqlc.narg('language'), language),
  avatar_url = COALESCE(sqlc.narg('avatar_url'), avatar_url),
  updated_at = now()
WHERE id = $1;

-- name: ListSaved :many
SELECT d.id, d.slug, d.name, d.district, d.altitude_m, d.rating, s.saved_at
FROM saved_destinations s
JOIN destinations d ON d.id = s.destination_id
WHERE s.user_id = $1
ORDER BY s.saved_at DESC;

-- name: SaveDestination :exec
INSERT INTO saved_destinations (user_id, destination_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: UnsaveDestination :exec
DELETE FROM saved_destinations WHERE user_id = $1 AND destination_id = $2;

-- name: ListItineraries :many
SELECT id, title, duration, start_date, is_public, share_token, created_at
FROM itineraries
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: CreateItinerary :one
INSERT INTO itineraries (user_id, title, duration, start_date, is_public)
VALUES ($1, $2, $3, $4, COALESCE(sqlc.narg('is_public')::bool, false))
RETURNING id, title, duration, start_date, is_public, share_token, created_at;
