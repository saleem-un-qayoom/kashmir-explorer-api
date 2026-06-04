-- 0014 · User-generated reviews + ratings for destinations and treks.
-- The destinations/treks rating + review_count columns are recomputed from
-- visible (non-hidden) reviews in the review handlers.

-- +goose Up
-- +goose StatementBegin
CREATE TABLE reviews (
  id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_type TEXT NOT NULL CHECK (target_type IN ('destination', 'trek')),
  target_id   UUID NOT NULL,                       -- destinations.id or treks.id (polymorphic)
  rating      INT  NOT NULL CHECK (rating BETWEEN 1 AND 5),
  body        TEXT,
  photos      TEXT[],
  hidden      BOOLEAN NOT NULL DEFAULT false,      -- admin moderation
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, target_type, target_id)         -- one review per user per target
);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX idx_reviews_target ON reviews(target_type, target_id) WHERE hidden = false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS reviews;
-- +goose StatementEnd
