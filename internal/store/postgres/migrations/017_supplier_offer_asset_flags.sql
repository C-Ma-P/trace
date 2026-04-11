-- +goose Up

alter table saved_supplier_offers
    add column has_symbol boolean not null default false,
    add column has_footprint boolean not null default false,
    add column has_datasheet boolean not null default false;

-- +goose Down

alter table saved_supplier_offers
    drop column has_symbol,
    drop column has_footprint,
    drop column has_datasheet;
