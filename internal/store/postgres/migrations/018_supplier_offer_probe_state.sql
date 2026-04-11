-- +goose Up

alter table saved_supplier_offers
    add column asset_probe_state text not null default 'unknown',
    add column asset_probe_error text not null default '',
    add column probe_completed_at timestamptz;

-- +goose Down

alter table saved_supplier_offers
    drop column probe_completed_at,
    drop column asset_probe_error,
    drop column asset_probe_state;
