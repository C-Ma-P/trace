-- +goose Up
alter table projects
    add column if not exists import_source_type text,
    add column if not exists import_source_path text,
    add column if not exists imported_at timestamptz;

-- +goose Down
alter table projects
    drop column if exists imported_at,
    drop column if exists import_source_path,
    drop column if exists import_source_type;
