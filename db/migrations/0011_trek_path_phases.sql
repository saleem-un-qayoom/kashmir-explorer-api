-- ────────────────────────────────────────────────────────────────────
-- 0011 · Multi-day trek path phases
-- ────────────────────────────────────────────────────────────────────
-- Treks were previously stored as a single polyline (path_geojson).
-- For multi-day routes (Kashmir Great Lakes, Tarsar-Marsar, Amarnath
-- Yatra) we want each day to be its own segment so the mobile app can
-- color-code days, the day-by-day card can highlight just that day,
-- and AMS planning knows exactly what altitude is gained per day.
--
-- New column: `path_phases JSONB` — an array of
--   [{ "day": 1, "coordinates": [[lng, lat], …] }, …]
--
-- `path_geojson` stays as the flat fallback so legacy clients still work.
-- ────────────────────────────────────────────────────────────────────

-- +goose Up
-- +goose StatementBegin

ALTER TABLE treks
  ADD COLUMN IF NOT EXISTS path_phases JSONB DEFAULT '[]'::jsonb;

-- Backfill: every existing single-line trek becomes a Day 1 phase.
UPDATE treks
   SET path_phases = jsonb_build_array(
         jsonb_build_object('day', 1, 'coordinates', path_geojson)
       )
 WHERE path_phases IS NULL
    OR path_phases = '[]'::jsonb
    AND jsonb_typeof(path_geojson) = 'array'
    AND jsonb_array_length(path_geojson) >= 2;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE treks DROP COLUMN IF EXISTS path_phases;
-- +goose StatementEnd
