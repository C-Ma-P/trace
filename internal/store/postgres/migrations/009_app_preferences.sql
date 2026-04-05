-- +goose Up
create table if not exists app_preferences (
    name text primary key,
    value_text text not null default '',
    updated_at timestamptz not null default now()
);

-- +goose Down
drop table if exists app_preferences;