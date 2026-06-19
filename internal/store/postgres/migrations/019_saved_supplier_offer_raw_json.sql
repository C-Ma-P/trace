-- +goose Up

alter table saved_supplier_offers
    add column raw_json jsonb;

-- +goose Down

alter table saved_supplier_offers
    drop column raw_json;