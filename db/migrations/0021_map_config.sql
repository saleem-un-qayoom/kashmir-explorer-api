-- 0021 · App-wide map configuration. A single-row table choosing which 3D map
-- engine the mobile app renders (CesiumJS vs Mapbox GL JS) plus a shared
-- terrain-exaggeration factor. The admin edits it; the mobile app fetches it at
-- launch and picks the matching map component.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE map_config (
  id                    INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),  -- single row
  engine                TEXT NOT NULL DEFAULT 'cesium'
                          CHECK (engine IN ('cesium', 'mapbox')),
  terrain_exaggeration  REAL NOT NULL DEFAULT 1.5
                          CHECK (terrain_exaggeration >= 0 AND terrain_exaggeration <= 3),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd
-- +goose StatementBegin
INSERT INTO map_config (id) VALUES (1) ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS map_config;
-- +goose StatementEnd
