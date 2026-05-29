-- ────────────────────────────────────────────────────────────────────
-- 0008 · Trail Sections (camps / stops per day)
-- ────────────────────────────────────────────────────────────────────
-- Adds a `trail_sections` JSONB column to the treks table so that
-- admins can define per-day / per-stop data: name, coordinates,
-- altitudes, distance, duration, description, difficulty, photos.
-- ────────────────────────────────────────────────────────────────────

-- +goose Up
-- +goose StatementBegin

ALTER TABLE treks
  ADD COLUMN IF NOT EXISTS trail_sections JSONB DEFAULT '[]'::jsonb;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE treks DROP COLUMN IF EXISTS trail_sections;

-- +goose StatementEnd
