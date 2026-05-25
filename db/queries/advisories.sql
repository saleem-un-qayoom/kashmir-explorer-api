-- name: ListAdvisories :many
SELECT id, severity, category, title, body, source, affected,
       confidence, effective_to, created_at
FROM advisories
WHERE effective_to > now()
  AND (sqlc.narg('severity')::text IS NULL OR severity = sqlc.narg('severity')::text)
  AND (sqlc.narg('category')::text IS NULL OR category = sqlc.narg('category')::text)
ORDER BY CASE severity WHEN 'critical' THEN 0 WHEN 'warning' THEN 1 ELSE 2 END,
         created_at DESC;

-- name: AdvisoriesForDestination :many
SELECT id, severity, category, title, body, source, affected,
       confidence, effective_to, created_at
FROM advisories
WHERE effective_to > now()
  AND ((scope = 'destination' AND scope_id = $1) OR scope = 'region')
ORDER BY CASE severity WHEN 'critical' THEN 0 WHEN 'warning' THEN 1 ELSE 2 END;

-- name: CreateAdvisory :one
INSERT INTO advisories
  (severity, category, title, body, source, affected, confidence,
   effective_from, effective_to)
VALUES
  ($1, $2, $3, $4, $5, $6, $7, now(), now() + ($8 || ' hours')::interval)
RETURNING id, severity, category, title, body, source, affected,
          confidence, effective_to, created_at;

-- name: ExpireOldAdvisories :execrows
DELETE FROM advisories WHERE effective_to <= now() - INTERVAL '1 hour';

-- name: ListRoadStatus :many
SELECT id, name, slug, current_status, closure_reason, last_checked
FROM roads
ORDER BY name;

-- name: UpdateRoadStatus :exec
UPDATE roads
SET current_status = $2, closure_reason = $3, last_checked = now()
WHERE id = $1;
