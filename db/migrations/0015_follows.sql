-- 0015 · Social graph: user follows. Powers public profiles + the activity
-- feed (reviews & summit completions from people you follow).

-- +goose Up
-- +goose StatementBegin
CREATE TABLE follows (
  follower_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  following_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (follower_id, following_id),
  CHECK (follower_id <> following_id)
);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE INDEX idx_follows_following ON follows(following_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS follows;
-- +goose StatementEnd
