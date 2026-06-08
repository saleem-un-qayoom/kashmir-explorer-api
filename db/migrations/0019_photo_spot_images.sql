-- Allow images to be attached to photo spots (in addition to destinations/treks).

-- +goose Up
-- +goose StatementBegin
ALTER TABLE images ADD COLUMN photo_spot_id UUID REFERENCES photo_spots(id) ON DELETE CASCADE;
CREATE INDEX idx_images_photo_spot ON images(photo_spot_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_images_photo_spot;
ALTER TABLE images DROP COLUMN IF EXISTS photo_spot_id;
-- +goose StatementEnd
