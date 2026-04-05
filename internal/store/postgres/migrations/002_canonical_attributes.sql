-- +goose Up
-- Add display_name and unit columns to attribute_definitions.
-- Canonical rows are populated at runtime by service.SyncCanonicalAttributeDefinitions(),
-- not by SQL seed data.
alter table attribute_definitions
    add column if not exists display_name text not null default '',
    add column if not exists unit text;

-- +goose Down
alter table attribute_definitions
    drop column if exists unit,
    drop column if exists display_name;
