-- +goose Up
alter table project_supplier_offer_snapshots rename to saved_supplier_offers;
alter table saved_supplier_offers rename column manufacturer_part_number to mpn;

-- +goose Down
alter table saved_supplier_offers rename column mpn to manufacturer_part_number;
alter table saved_supplier_offers rename to project_supplier_offer_snapshots;
