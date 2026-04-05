-- +goose Up
drop table if exists inventory_lots;

-- +goose Down
create table if not exists inventory_lots (
    id text primary key,
    component_id text not null references components(id) on delete cascade,
    quantity integer not null,
    location text not null default '',
    supplier text not null default '',
    created_at timestamptz not null default now()
);
create index if not exists idx_inventory_lots_component_id on inventory_lots(component_id);
