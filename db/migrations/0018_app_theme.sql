-- 0018 · App-wide theme overrides. A single-row table holding color overrides
-- for the mobile app's design tokens (palette keys → hex). The admin edits it;
-- the mobile app fetches it at launch and applies the overrides on top of the
-- built-in palette. Empty `{}` means "use the shipped defaults".

-- +goose Up
-- +goose StatementBegin
CREATE TABLE app_theme (
  id         INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),  -- enforce a single row
  colors     JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd
-- +goose StatementBegin
INSERT INTO app_theme (id, colors) VALUES (1, '{}'::jsonb) ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS app_theme;
-- +goose StatementEnd
