-- Explicit admin toggles for whether a destination requires a permit and
-- whether it charges an entry fee. The mobile UI only renders the permit pill
-- and the entry-fee stat when these are true (e.g. Dal Lake needs neither).

-- +goose Up
-- +goose StatementBegin
ALTER TABLE destinations ADD COLUMN requires_permit BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE destinations ADD COLUMN has_entry_fee   BOOLEAN NOT NULL DEFAULT false;

-- Backfill from existing data so current rows stay consistent.
UPDATE destinations SET requires_permit = true
  WHERE permits IS NOT NULL AND array_length(permits, 1) > 0;
UPDATE destinations SET has_entry_fee = true
  WHERE entry_fee_inr IS NOT NULL AND entry_fee_inr > 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE destinations DROP COLUMN IF EXISTS has_entry_fee;
ALTER TABLE destinations DROP COLUMN IF EXISTS requires_permit;
-- +goose StatementEnd
