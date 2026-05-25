-- +goose Up
-- +goose StatementBegin

-- ─── pgvector embeddings ─────────────────────────────────────
-- Voyage AI 'voyage-3-lite' is 512-dim. We re-create the column at the right
-- size and add an HNSW index for fast cosine-similarity search.
ALTER TABLE destinations DROP COLUMN IF EXISTS embedding;
ALTER TABLE destinations ADD COLUMN embedding vector(512);
CREATE INDEX IF NOT EXISTS destinations_embedding_hnsw
  ON destinations USING hnsw (embedding vector_cosine_ops);

ALTER TABLE treks ADD COLUMN IF NOT EXISTS embedding vector(512);
CREATE INDEX IF NOT EXISTS treks_embedding_hnsw
  ON treks USING hnsw (embedding vector_cosine_ops);

-- ─── Crowdsourced reports on treks ────────────────────────────
CREATE TABLE IF NOT EXISTS trek_reports (
  id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  trek_id       UUID NOT NULL REFERENCES treks(id) ON DELETE CASCADE,
  user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
  category      TEXT NOT NULL,                  -- 'wrong_path' | 'blocked' | 'unsafe' | 'wildlife' | 'other'
  body          TEXT,
  location      GEOGRAPHY(POINT, 4326),         -- where the issue was spotted
  status        TEXT NOT NULL DEFAULT 'open',   -- 'open' | 'reviewing' | 'resolved' | 'dismissed'
  admin_note    TEXT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  resolved_at   TIMESTAMPTZ
);
CREATE INDEX trek_reports_trek_status ON trek_reports(trek_id, status);
CREATE INDEX trek_reports_location    ON trek_reports USING GIST(location);

-- ─── Crowd density (active nav pings) ────────────────────────
-- We keep an append-only log of who's actively navigating which trek, with
-- TTL via partial index — purged hourly by the alert-worker job.
CREATE TABLE IF NOT EXISTS trek_nav_pings (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  trek_slug   TEXT NOT NULL,
  user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
  along_m     INT,                              -- progress along polyline
  recorded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX trek_nav_pings_recent
  ON trek_nav_pings(trek_slug, recorded_at DESC)
;

-- ─── Group trips (real-time location share between friends) ──
CREATE TABLE IF NOT EXISTS trip_groups (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  owner_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name        TEXT NOT NULL,
  invite_code TEXT UNIQUE NOT NULL,             -- 6-char shareable code
  trek_slug   TEXT,                             -- optional trek context
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at  TIMESTAMPTZ NOT NULL DEFAULT (now() + INTERVAL '7 days')
);

CREATE TABLE IF NOT EXISTS trip_group_members (
  group_id    UUID NOT NULL REFERENCES trip_groups(id) ON DELETE CASCADE,
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  joined_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (group_id, user_id)
);

-- ─── Premium subscriptions ───────────────────────────────────
CREATE TABLE IF NOT EXISTS subscriptions (
  id                       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id                  UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
  plan                     TEXT NOT NULL,        -- 'monthly' | 'yearly'
  status                   TEXT NOT NULL,        -- 'active' | 'past_due' | 'cancelled' | 'expired'
  razorpay_subscription_id TEXT,
  razorpay_customer_id     TEXT,
  current_period_end       TIMESTAMPTZ,
  cancelled_at             TIMESTAMPTZ,
  created_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE users ADD COLUMN IF NOT EXISTS is_premium BOOLEAN DEFAULT FALSE;

-- ─── Wallet pass tracking ────────────────────────────────────
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS wallet_serial TEXT;
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS wallet_auth_token TEXT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE bookings DROP COLUMN IF EXISTS wallet_auth_token, DROP COLUMN IF EXISTS wallet_serial;
ALTER TABLE users    DROP COLUMN IF EXISTS is_premium;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS trip_group_members;
DROP TABLE IF EXISTS trip_groups;
DROP TABLE IF EXISTS trek_nav_pings;
DROP TABLE IF EXISTS trek_reports;
DROP INDEX IF EXISTS treks_embedding_hnsw;
DROP INDEX IF EXISTS destinations_embedding_hnsw;
ALTER TABLE treks        DROP COLUMN IF EXISTS embedding;
ALTER TABLE destinations DROP COLUMN IF EXISTS embedding;
-- +goose StatementEnd
