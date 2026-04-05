-- +goose Up

create table if not exists project_part_candidates (
    id text primary key,
    project_id text not null references projects(id) on delete cascade,
    requirement_id text not null references project_requirements(id) on delete cascade,
    component_id text not null references components(id) on delete cascade,
    preferred boolean not null default false,
    origin text not null default 'local',
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique(requirement_id, component_id)
);

create unique index if not exists idx_part_candidates_preferred
    on project_part_candidates(requirement_id) where preferred = true;

create index if not exists idx_part_candidates_requirement
    on project_part_candidates(requirement_id);

create index if not exists idx_part_candidates_project
    on project_part_candidates(project_id);

create table if not exists saved_supplier_offers (
    id text primary key,
    project_id text not null references projects(id) on delete cascade,
    requirement_id text not null references project_requirements(id) on delete cascade,
    provider text not null,
    provider_part_id text not null default '',
    product_url text not null default '',
    manufacturer text not null default '',
    mpn text not null default '',
    description text not null default '',
    package text not null default '',
    stock integer,
    moq integer,
    unit_price double precision,
    currency text not null default 'USD',
    linked_component_id text references components(id) on delete set null,
    captured_at timestamptz not null default now(),
    created_at timestamptz not null default now()
);

create index if not exists idx_saved_offers_requirement
    on saved_supplier_offers(requirement_id);

create index if not exists idx_saved_offers_project
    on saved_supplier_offers(project_id);

-- +goose Down
drop table if exists saved_supplier_offers;
drop table if exists project_part_candidates;
