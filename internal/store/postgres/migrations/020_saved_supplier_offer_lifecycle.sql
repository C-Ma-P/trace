-- +goose Up

alter table saved_supplier_offers
    add column lifecycle text not null default '';

-- +goose Down

alter table saved_supplier_offers
    drop column lifecycle;