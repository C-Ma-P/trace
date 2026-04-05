-- +goose Up

-- Allow candidates without a component (provider-backed, not yet imported).
alter table project_part_candidates alter column component_id drop not null;

-- Track which saved offer backs a provider-backed candidate.
alter table project_part_candidates
    add column source_offer_id text references saved_supplier_offers(id) on delete set null;

-- Replace the old unique constraint with partial indexes that handle NULLs correctly.
alter table project_part_candidates
    drop constraint if exists project_part_candidates_requirement_id_component_id_key;

create unique index if not exists idx_part_candidates_req_component
    on project_part_candidates(requirement_id, component_id) where component_id is not null;

create unique index if not exists idx_part_candidates_req_offer
    on project_part_candidates(requirement_id, source_offer_id) where source_offer_id is not null;

-- +goose Down
drop index if exists idx_part_candidates_req_offer;
drop index if exists idx_part_candidates_req_component;
alter table project_part_candidates drop column if exists source_offer_id;
alter table project_part_candidates alter column component_id set not null;
alter table project_part_candidates add constraint project_part_candidates_requirement_id_component_id_key unique(requirement_id, component_id);
