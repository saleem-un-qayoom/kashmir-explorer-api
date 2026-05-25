-- +goose Up
-- +goose StatementBegin

-- Extensions
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ─── Regions
CREATE TABLE regions (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name        TEXT NOT NULL,
  slug        TEXT UNIQUE NOT NULL,
  description TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ─── Categories
CREATE TABLE categories (
  id    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name  TEXT NOT NULL,
  slug  TEXT UNIQUE NOT NULL,
  icon  TEXT,
  color TEXT
);

-- ─── Destinations
CREATE TABLE destinations (
  id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name                  TEXT NOT NULL,
  name_urdu             TEXT,
  name_hindi            TEXT,
  slug                  TEXT UNIQUE NOT NULL,
  region_id             UUID REFERENCES regions(id) ON DELETE SET NULL,
  district              TEXT,
  tehsil                TEXT,
  tagline               TEXT,
  description           TEXT,
  uniqueness            TEXT,
  location              GEOGRAPHY(POINT, 4326),
  address               TEXT,
  altitude_m            INT,
  best_months           INT[],
  season_type           TEXT,
  rating                NUMERIC(3,2) DEFAULT 0,
  review_count          INT DEFAULT 0,
  distance_from_srinagar_km INT,
  entry_fee_inr         INT DEFAULT 0,
  entry_fee_foreign_inr INT,
  network_coverage      JSONB,
  practical             JSONB,
  permits               TEXT[],
  open_hours            JSONB,
  closure_dates         DATE[],
  emergency_contacts    JSONB,
  embedding             vector(1536),
  is_published          BOOLEAN NOT NULL DEFAULT false,
  is_featured           BOOLEAN NOT NULL DEFAULT false,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_destinations_location ON destinations USING GIST(location);
CREATE INDEX idx_destinations_district ON destinations(district);
CREATE INDEX idx_destinations_slug ON destinations(slug);
CREATE INDEX idx_destinations_published ON destinations(is_published) WHERE is_published = true;

-- Destination ↔ category m2m
CREATE TABLE destination_categories (
  destination_id UUID NOT NULL REFERENCES destinations(id) ON DELETE CASCADE,
  category_id    UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
  PRIMARY KEY (destination_id, category_id)
);

-- Activities (loose tags per destination)
CREATE TABLE destination_activities (
  destination_id UUID NOT NULL REFERENCES destinations(id) ON DELETE CASCADE,
  activity       TEXT NOT NULL,
  PRIMARY KEY (destination_id, activity)
);

-- ─── Images
CREATE TABLE images (
  id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  destination_id UUID REFERENCES destinations(id) ON DELETE CASCADE,
  trek_id        UUID,
  url            TEXT NOT NULL,
  blurhash       TEXT,
  caption        TEXT,
  is_hero        BOOLEAN DEFAULT false,
  sort_order     INT DEFAULT 0,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_images_destination ON images(destination_id);

-- ─── Treks (extends destination)
CREATE TABLE treks (
  id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  slug              TEXT UNIQUE NOT NULL,
  name              TEXT NOT NULL,
  destination_id    UUID REFERENCES destinations(id) ON DELETE SET NULL,
  difficulty        TEXT NOT NULL,        -- easy | moderate | hard | strenuous
  trek_type         TEXT NOT NULL,        -- meadow | alpine_lake | glacier | pass | valley
  duration_days     INT NOT NULL,
  distance_km       NUMERIC(6,2),
  max_altitude_m    INT,
  start_point       TEXT,
  end_point         TEXT,
  start_location    GEOGRAPHY(POINT, 4326),
  waypoints         JSONB,                -- [{lat,lng,name,day,type,altitude_m,notes}]
  gear_list         JSONB,                -- [{name,category,essential}]
  best_months       INT[],
  permits           TEXT[],
  permit_details    TEXT,
  guide_available   BOOLEAN DEFAULT true,
  guide_price_inr   INT,
  ams_risk          BOOLEAN DEFAULT false,
  status            TEXT DEFAULT 'closed', -- open | closing-soon | closed
  closure_reason    TEXT,
  tagline           TEXT,
  uniqueness        TEXT,
  rating            NUMERIC(3,2) DEFAULT 0,
  review_count      INT DEFAULT 0,
  is_published      BOOLEAN NOT NULL DEFAULT false,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_treks_start_location ON treks USING GIST(start_location);
CREATE INDEX idx_treks_slug ON treks(slug);

-- ─── Users
CREATE TABLE users (
  id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name         TEXT,
  email        TEXT UNIQUE,
  phone        TEXT UNIQUE,
  avatar_url   TEXT,
  provider     TEXT,                       -- 'google' | 'apple' | 'phone'
  provider_id  TEXT,
  role         TEXT NOT NULL DEFAULT 'user', -- 'user' | 'admin'
  language     TEXT DEFAULT 'en',
  medical      JSONB,
  insurance    JSONB,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ─── OTP store
CREATE TABLE otp_codes (
  phone      TEXT NOT NULL,
  code_hash  TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  attempts   INT NOT NULL DEFAULT 0,
  PRIMARY KEY (phone)
);

-- ─── Saved (wishlist)
CREATE TABLE saved_destinations (
  user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  destination_id UUID NOT NULL REFERENCES destinations(id) ON DELETE CASCADE,
  saved_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, destination_id)
);

-- ─── Itineraries
CREATE TABLE itineraries (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title       TEXT NOT NULL,
  duration    INT,
  start_date  DATE,
  is_public   BOOLEAN NOT NULL DEFAULT false,
  share_token TEXT UNIQUE,
  metadata    JSONB,                 -- persona, budget, constraints
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE itinerary_days (
  id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  itinerary_id  UUID NOT NULL REFERENCES itineraries(id) ON DELETE CASCADE,
  day_number    INT NOT NULL,
  title         TEXT,
  notes         TEXT,
  UNIQUE (itinerary_id, day_number)
);

CREATE TABLE itinerary_stops (
  id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  day_id         UUID NOT NULL REFERENCES itinerary_days(id) ON DELETE CASCADE,
  destination_id UUID REFERENCES destinations(id),
  trek_id        UUID REFERENCES treks(id),
  sort_order     INT NOT NULL DEFAULT 0,
  notes          TEXT
);

-- ─── Advisories (real-time alerts)
CREATE TABLE advisories (
  id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  scope          TEXT,                 -- 'region' | 'destination' | 'trek' | 'road'
  scope_id       UUID,
  severity       TEXT NOT NULL,        -- 'critical' | 'warning' | 'info'
  category       TEXT NOT NULL,        -- 'weather' | 'road' | 'avalanche' | 'security' | 'health'
  title          TEXT NOT NULL,
  body           TEXT,
  source         TEXT,                 -- 'IMD' | 'JKTDC' | 'NDMA' | 'admin'
  source_url     TEXT,
  affected       TEXT,
  confidence     INT NOT NULL DEFAULT 100,
  effective_from TIMESTAMPTZ,
  effective_to   TIMESTAMPTZ,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_advisories_active ON advisories(severity, effective_to)
  WHERE effective_to IS NOT NULL;

-- ─── Roads (Zojila, NH-44, Mughal Road)
CREATE TABLE roads (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name            TEXT NOT NULL,
  slug            TEXT UNIQUE NOT NULL,
  geom            GEOGRAPHY(LINESTRING, 4326),
  current_status  TEXT NOT NULL DEFAULT 'open', -- 'open' | 'one-way' | 'closed' | 'restricted'
  last_checked    TIMESTAMPTZ NOT NULL DEFAULT now(),
  closure_reason  TEXT,
  alternate_id    UUID REFERENCES roads(id)
);

-- ─── Weather snapshots
CREATE TABLE weather_snapshots (
  destination_id UUID NOT NULL REFERENCES destinations(id) ON DELETE CASCADE,
  fetched_at     TIMESTAMPTZ NOT NULL,
  temp_c         NUMERIC(4,1),
  feels_like_c   NUMERIC(4,1),
  condition      TEXT,
  icon           TEXT,
  wind_kmh       NUMERIC(5,1),
  precip_mm      NUMERIC(5,1),
  humidity_pct   INT,
  visibility_km  NUMERIC(5,1),
  aqi            INT,
  sunrise        TIME,
  sunset         TIME,
  source         TEXT,
  PRIMARY KEY (destination_id, fetched_at)
);

-- ─── Photo spots
CREATE TABLE photo_spots (
  id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  destination_id      UUID NOT NULL REFERENCES destinations(id) ON DELETE CASCADE,
  name                TEXT NOT NULL,
  location            GEOGRAPHY(POINT,4326),
  best_time           TEXT,
  facing              TEXT,
  difficulty_to_reach TEXT,
  tripod_recommended  BOOLEAN DEFAULT false,
  drone_allowed       BOOLEAN DEFAULT false,
  description         TEXT
);

-- ─── Providers (houseboats, shikara, guides, ponies, cabs, helis)
CREATE TABLE providers (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  type            TEXT NOT NULL,
  name            TEXT NOT NULL,
  jktdc_reg_no    TEXT,
  verified        BOOLEAN NOT NULL DEFAULT false,
  base_location   GEOGRAPHY(POINT,4326),
  base_location_text TEXT,
  languages       TEXT[],
  rating          NUMERIC(3,2) DEFAULT 0,
  review_count    INT DEFAULT 0,
  phone           TEXT,
  whatsapp        TEXT,
  capacity        INT,
  amenities       TEXT[],
  price_inr       INT NOT NULL,
  price_unit      TEXT NOT NULL,    -- 'per-night' | 'per-trip' | 'per-day' | 'per-hour'
  cancellation    TEXT,
  description     TEXT,
  documents       JSONB,
  years_hosting   INT,
  response_time_min INT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_providers_type ON providers(type);
CREATE INDEX idx_providers_verified ON providers(verified) WHERE verified = true;

-- ─── Bookings
CREATE TABLE bookings (
  id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  ref                 TEXT UNIQUE NOT NULL,             -- KEX-49813 style
  user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider_id         UUID NOT NULL REFERENCES providers(id),
  start_date          DATE NOT NULL,
  end_date            DATE,
  guests              INT NOT NULL DEFAULT 1,
  base_inr            INT NOT NULL,
  gst_inr             INT NOT NULL,
  fee_inr             INT NOT NULL,
  total_inr           INT NOT NULL,
  status              TEXT NOT NULL DEFAULT 'pending',  -- pending|confirmed|cancelled|completed|refunded
  razorpay_order_id   TEXT,
  razorpay_payment_id TEXT,
  notes               TEXT,
  cancellation_reason TEXT,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ─── Sync queue (offline mutations)
CREATE TABLE sync_queue (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  op          TEXT NOT NULL,
  payload     JSONB,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  applied_at  TIMESTAMPTZ
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS sync_queue;
DROP TABLE IF EXISTS bookings;
DROP TABLE IF EXISTS providers;
DROP TABLE IF EXISTS photo_spots;
DROP TABLE IF EXISTS weather_snapshots;
DROP TABLE IF EXISTS roads;
DROP TABLE IF EXISTS advisories;
DROP TABLE IF EXISTS itinerary_stops;
DROP TABLE IF EXISTS itinerary_days;
DROP TABLE IF EXISTS itineraries;
DROP TABLE IF EXISTS saved_destinations;
DROP TABLE IF EXISTS otp_codes;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS treks;
DROP TABLE IF EXISTS images;
DROP TABLE IF EXISTS destination_activities;
DROP TABLE IF EXISTS destination_categories;
DROP TABLE IF EXISTS destinations;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS regions;

DROP EXTENSION IF EXISTS vector;
DROP EXTENSION IF EXISTS pg_trgm;
DROP EXTENSION IF EXISTS postgis;

-- +goose StatementEnd
