-- ────────────────────────────────────────────────────────────────────
-- 0007 · Trek Companion V3
-- ────────────────────────────────────────────────────────────────────
-- Extends the existing `trek_reports` table for the V3 community
-- trail-condition flow (severity, photo, waypoint) and adds two new
-- tables: track_recordings (saved GPX hikes) + summit_completions
-- (the "bagged" / summit-log wall).
-- ────────────────────────────────────────────────────────────────────

-- +goose Up
-- +goose StatementBegin

-- ── trek_reports: V3 columns ────────────────────────────────────────
ALTER TABLE trek_reports
  ADD COLUMN IF NOT EXISTS severity      INT,             -- 1..5
  ADD COLUMN IF NOT EXISTS photo_url     TEXT,            -- optional R2 url
  ADD COLUMN IF NOT EXISTS waypoint_idx  INT,             -- 0-based on trek polyline
  ADD COLUMN IF NOT EXISTS expires_at    TIMESTAMPTZ;     -- auto-hide stale reports

-- Default severity for legacy rows
UPDATE trek_reports SET severity = 3 WHERE severity IS NULL;

-- Default expires_at to created_at + 14 days for active reports (V3 spec)
UPDATE trek_reports
   SET expires_at = created_at + INTERVAL '14 days'
 WHERE expires_at IS NULL AND status IN ('open','reviewing');

CREATE INDEX IF NOT EXISTS idx_trek_reports_active
  ON trek_reports(trek_id, expires_at)
  WHERE status IN ('open','reviewing');

-- ── track_recordings: saved GPX hikes ──────────────────────────────
CREATE TABLE IF NOT EXISTS track_recordings (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  trek_id         UUID REFERENCES treks(id) ON DELETE SET NULL,
  name            TEXT NOT NULL,
  started_at      TIMESTAMPTZ NOT NULL,
  ended_at        TIMESTAMPTZ,
  distance_m      INT  NOT NULL DEFAULT 0,
  duration_s      INT  NOT NULL DEFAULT 0,
  gain_m          INT  NOT NULL DEFAULT 0,
  loss_m          INT  NOT NULL DEFAULT 0,
  max_altitude_m  INT,
  /** Array of [lng, lat, ts_ms, altitude?] */
  polyline        JSONB NOT NULL DEFAULT '[]'::jsonb,
  /** Token for /track/share — public read URL. */
  share_token     TEXT UNIQUE,
  is_public       BOOLEAN NOT NULL DEFAULT false,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_track_recordings_user
  ON track_recordings(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_track_recordings_share
  ON track_recordings(share_token) WHERE share_token IS NOT NULL;

-- ── summit_completions: "bagged" peaks / treks ─────────────────────
CREATE TABLE IF NOT EXISTS summit_completions (
  id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  /** Exactly one of these is set. */
  trek_id            UUID REFERENCES treks(id) ON DELETE CASCADE,
  destination_id     UUID REFERENCES destinations(id) ON DELETE CASCADE,
  completed_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  /** Link to the recording that proves it. Optional — users can mark
   *  manually without a recording. */
  track_recording_id UUID REFERENCES track_recordings(id) ON DELETE SET NULL,
  notes              TEXT,
  CHECK ( (trek_id IS NULL) <> (destination_id IS NULL) )
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_summit_completions_unique_trek
  ON summit_completions(user_id, trek_id) WHERE trek_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_summit_completions_unique_dest
  ON summit_completions(user_id, destination_id) WHERE destination_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS summit_completions;
DROP TABLE IF EXISTS track_recordings;
ALTER TABLE trek_reports
  DROP COLUMN IF EXISTS severity,
  DROP COLUMN IF EXISTS photo_url,
  DROP COLUMN IF EXISTS waypoint_idx,
  DROP COLUMN IF EXISTS expires_at;
-- +goose StatementEnd
