-- Explicit admin toggle for whether a trek requires a permit. Mirrors the
-- destinations flag; the mobile UI only renders the permit note/pill when true.

-- +goose Up
-- +goose StatementBegin
ALTER TABLE treks ADD COLUMN requires_permit BOOLEAN NOT NULL DEFAULT false;

-- Backfill from existing data so current rows stay consistent.
UPDATE treks SET requires_permit = true
  WHERE permits IS NOT NULL AND array_length(permits, 1) > 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE treks DROP COLUMN IF EXISTS requires_permit;
-- +goose StatementEnd
