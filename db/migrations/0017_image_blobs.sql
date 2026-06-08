-- 0017 · Store image bytes directly in the DB (interim, "for now"). Until
-- object storage (Supabase/R2) is wired up, uploaded images live in the
-- `images` table as BYTEA and are served back via /v1/images/{id}/raw.
-- `url` becomes nullable: blob-backed rows derive their URL from the id.

-- +goose Up
-- +goose StatementBegin
ALTER TABLE images
  ADD COLUMN data         BYTEA,
  ADD COLUMN content_type TEXT,
  ALTER COLUMN url DROP NOT NULL;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE images
  ADD CONSTRAINT images_url_or_data CHECK (url IS NOT NULL OR data IS NOT NULL);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE images DROP CONSTRAINT IF EXISTS images_url_or_data;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE images
  DROP COLUMN IF EXISTS data,
  DROP COLUMN IF EXISTS content_type,
  ALTER COLUMN url SET NOT NULL;
-- +goose StatementEnd
