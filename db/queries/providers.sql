-- name: ListProviders :many
SELECT id, type, name, jktdc_reg_no, verified, base_location_text,
       languages, rating, review_count, capacity, amenities, price_inr, price_unit,
       cancellation, description, years_hosting, response_time_min, phone, whatsapp
FROM providers
WHERE (sqlc.narg('type')::text IS NULL OR type = sqlc.narg('type')::text)
  AND (NOT sqlc.arg('verified_only')::bool OR verified = true)
ORDER BY verified DESC, rating DESC;

-- name: GetProvider :one
SELECT id, type, name, jktdc_reg_no, verified, base_location_text,
       languages, rating, review_count, capacity, amenities, price_inr, price_unit,
       cancellation, description, years_hosting, response_time_min, phone, whatsapp
FROM providers
WHERE id = $1;

-- name: VerifyProvider :exec
UPDATE providers SET verified = true WHERE id = $1;
