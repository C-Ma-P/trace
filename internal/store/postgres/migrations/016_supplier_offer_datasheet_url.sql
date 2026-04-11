-- +goose Up

ALTER TABLE saved_supplier_offers ADD COLUMN datasheet_url text NOT NULL DEFAULT '';

-- +goose Down

ALTER TABLE saved_supplier_offers DROP COLUMN datasheet_url;
