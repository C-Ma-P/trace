-- +goose Up
ALTER TABLE saved_supplier_offers ADD COLUMN image_url text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE saved_supplier_offers DROP COLUMN image_url;
