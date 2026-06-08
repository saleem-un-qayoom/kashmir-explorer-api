-- 0016 · Home-screen hero banners. A curated, ordered carousel managed in the
-- admin, independent of any single destination/trek. The mobile home screen
-- reads the active banners instead of hijacking the "featured" destination's
-- hero image.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE home_hero_banners (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  image_url   TEXT NOT NULL,
  blurhash    TEXT,
  title       TEXT,
  subtitle    TEXT,
  link_type   TEXT NOT NULL DEFAULT 'none',  -- none | destination | trek | screen
  link_value  TEXT,                          -- slug (destination/trek) or route path (screen)
  sort_order  INT  NOT NULL DEFAULT 0,
  is_active   BOOLEAN NOT NULL DEFAULT true,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX idx_home_hero_active ON home_hero_banners(is_active, sort_order);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS home_hero_banners;
-- +goose StatementEnd
