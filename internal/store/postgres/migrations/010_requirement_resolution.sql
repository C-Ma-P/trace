-- +goose Up
alter table project_requirements
    add column if not exists resolution_kind text,
    add column if not exists resolution_component_id text references components(id) on delete set null;

update project_requirements
set resolution_kind = 'internal_component',
    resolution_component_id = selected_component_id
where selected_component_id is not null
  and (resolution_kind is null or resolution_component_id is null);

-- +goose Down
alter table project_requirements
    drop column if exists resolution_component_id,
    drop column if exists resolution_kind;