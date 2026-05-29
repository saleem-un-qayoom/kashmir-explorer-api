-- ────────────────────────────────────────────────────────────────────
-- 0010 · Destinations gain AllTrails-style `features` tags
-- ────────────────────────────────────────────────────────────────────
-- Same filter-chip vocabulary as treks (kid_friendly, dog_friendly,
-- waterfall, wildflowers, wildlife, …) — so the mobile Explore screen
-- can filter destinations + treks with one shared UI.
-- ────────────────────────────────────────────────────────────────────

-- +goose Up
-- +goose StatementBegin
ALTER TABLE destinations
  ADD COLUMN IF NOT EXISTS features TEXT[] DEFAULT '{}'::TEXT[];
CREATE INDEX IF NOT EXISTS idx_destinations_features
  ON destinations USING gin(features);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE destinations DROP COLUMN IF EXISTS features;
-- +goose StatementEnd
