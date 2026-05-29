-- ────────────────────────────────────────────────────────────────────
-- 0008 · AllTrails-style trek fields
-- ────────────────────────────────────────────────────────────────────
-- Adds feature tags + activity types to treks so the mobile app and
-- the admin editor can power AllTrails-style filtering and discovery.
-- ────────────────────────────────────────────────────────────────────

-- +goose Up
-- +goose StatementBegin

ALTER TABLE treks
  ADD COLUMN IF NOT EXISTS features    TEXT[] DEFAULT '{}'::TEXT[],
  ADD COLUMN IF NOT EXISTS activities  TEXT[] DEFAULT '{hike}'::TEXT[],
  ADD COLUMN IF NOT EXISTS elevation_gain_m INT,
  ADD COLUMN IF NOT EXISTS route_type  TEXT;  -- 'out-and-back' | 'loop' | 'point-to-point'

-- Backfill route_type from existing trek_type-like inference. Best-effort:
-- treks where start_point = end_point → loop; otherwise out-and-back.
UPDATE treks
   SET route_type = CASE
     WHEN start_point IS NOT NULL AND start_point = end_point THEN 'loop'
     WHEN start_point IS NOT NULL AND end_point IS NOT NULL AND start_point <> end_point THEN 'point-to-point'
     ELSE 'out-and-back'
   END
 WHERE route_type IS NULL;

CREATE INDEX IF NOT EXISTS idx_treks_features   ON treks USING gin(features);
CREATE INDEX IF NOT EXISTS idx_treks_activities ON treks USING gin(activities);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE treks
  DROP COLUMN IF EXISTS features,
  DROP COLUMN IF EXISTS activities,
  DROP COLUMN IF EXISTS elevation_gain_m,
  DROP COLUMN IF EXISTS route_type;
-- +goose StatementEnd
