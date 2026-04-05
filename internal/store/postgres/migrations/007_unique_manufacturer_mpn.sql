-- +goose Up
-- Remove duplicate rows introduced by running seed multiple times,
-- keeping the earliest-created row for each (manufacturer, mpn) pair.
delete from components
where id in (
    select id from (
        select id,
               row_number() over (partition by manufacturer, mpn order by created_at asc) as rn
        from components
    ) dupes
    where rn > 1
);

alter table components
    add constraint components_manufacturer_mpn_key unique (manufacturer, mpn);

-- +goose Down
alter table components
    drop constraint if exists components_manufacturer_mpn_key;
