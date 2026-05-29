-- +goose Up
ALTER TABLE destinations ADD COLUMN is_deleted BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE destinations DROP COLUMN is_deleted;
