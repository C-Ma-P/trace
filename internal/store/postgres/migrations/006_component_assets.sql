-- +goose Up

-- Asset candidates table
create table if not exists component_assets (
    id text primary key,
    component_id text not null references components(id) on delete cascade,
    asset_type text not null check (asset_type in ('symbol', 'footprint', '3d_model', 'datasheet')),
    source text not null default 'manual',
    status text not null default 'candidate' check (status in ('candidate', 'selected', 'verified', 'rejected')),
    label text not null default '',
    url_or_path text not null default '',
    preview_url text,
    metadata_json jsonb,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index if not exists idx_component_assets_component_id on component_assets(component_id);
create index if not exists idx_component_assets_component_type on component_assets(component_id, asset_type);

-- Selected-asset pointers on components (nullable FKs).
-- ON DELETE SET NULL: if the selected asset row is deleted, the pointer clears automatically.
alter table components
    add column if not exists selected_symbol_asset_id text references component_assets(id) on delete set null,
    add column if not exists selected_footprint_asset_id text references component_assets(id) on delete set null,
    add column if not exists selected_3d_model_asset_id text references component_assets(id) on delete set null,
    add column if not exists selected_datasheet_asset_id text references component_assets(id) on delete set null;

-- +goose Down
alter table components
    drop column if exists selected_symbol_asset_id,
    drop column if exists selected_footprint_asset_id,
    drop column if exists selected_3d_model_asset_id,
    drop column if exists selected_datasheet_asset_id;

drop table if exists component_assets;
