-- +goose Up
CREATE TABLE inventory_bags (
    id text PRIMARY KEY,
    label text NOT NULL DEFAULT '',
    qr_data text NOT NULL,
    component_id text NOT NULL REFERENCES components(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_inventory_bags_qr_data ON inventory_bags(qr_data);
CREATE INDEX idx_inventory_bags_component_id ON inventory_bags(component_id);

-- +goose Down
DROP TABLE IF EXISTS inventory_bags;
