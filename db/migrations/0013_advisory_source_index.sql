-- 0012 · Unique index on advisories.source_url for external-data upserts.
-- The fetcher uses ON CONFLICT (source_url) to deduplicate NDMA/IMD entries.

-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS idx_advisories_source_url
  ON advisories(source_url)
  WHERE source_url IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_advisories_source_url;
-- +goose StatementEnd
