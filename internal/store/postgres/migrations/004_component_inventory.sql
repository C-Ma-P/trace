-- +goose Up
-- Add lightweight inventory fields directly to the components table.
-- Existing rows default to unknown mode with no quantity and empty location.
alter table components
    add column if not exists quantity      integer,
    add column if not exists quantity_mode text    not null default 'unknown',
    add column if not exists location      text    not null default '';

-- +goose Down
alter table components
    drop column if exists location,
    drop column if exists quantity_mode,
    drop column if exists quantity;
