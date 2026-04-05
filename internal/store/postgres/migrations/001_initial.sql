-- +goose Up
create extension if not exists pgcrypto;

create table if not exists attribute_definitions (
    key text not null,
    category text not null,
    value_type text not null,
    primary key (key, category)
);

create table if not exists components (
    id text primary key,
    category text not null,
    mpn text not null,
    manufacturer text not null,
    package text not null default '',
    description text not null default '',
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists component_attributes (
    component_id text not null references components(id) on delete cascade,
    key text not null,
    value_type text not null,
    text_value text,
    number_value double precision,
    bool_value boolean,
    unit text not null default '',
    primary key (component_id, key)
);

create table if not exists inventory_lots (
    id text primary key,
    component_id text not null references components(id) on delete cascade,
    quantity integer not null,
    location text not null default '',
    supplier text not null default '',
    created_at timestamptz not null default now()
);

create table if not exists projects (
    id text primary key,
    name text not null,
    description text not null default '',
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists project_requirements (
    id text primary key,
    project_id text not null references projects(id) on delete cascade,
    name text not null,
    category text not null,
    quantity integer not null,
    selected_component_id text references components(id) on delete set null
);

create table if not exists requirement_constraints (
    requirement_id text not null references project_requirements(id) on delete cascade,
    key text not null,
    value_type text not null,
    operator text not null,
    text_value text,
    number_value double precision,
    bool_value boolean,
    unit text not null default '',
    primary key (requirement_id, key, operator)
);

create index if not exists idx_components_category on components(category);
create index if not exists idx_inventory_lots_component_id on inventory_lots(component_id);
create index if not exists idx_project_requirements_project_id on project_requirements(project_id);

-- +goose Down
drop table if exists requirement_constraints;
drop table if exists project_requirements;
drop table if exists inventory_lots;
drop table if exists component_attributes;
drop table if exists components;
drop table if exists projects;
drop table if exists attribute_definitions;
drop extension if exists pgcrypto;
