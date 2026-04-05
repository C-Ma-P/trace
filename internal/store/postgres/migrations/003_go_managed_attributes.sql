-- +goose Up
-- Authority boundary marker: canonical attribute_definitions rows are owned by
-- the Go registry (internal/domain/registry) and written at startup via
-- service.SyncCanonicalAttributeDefinitions(). No SQL seeding occurs from this
-- point forward. Any rows previously seeded by migration 002 are superseded by
-- the sync on next startup.
delete from attribute_definitions where category in ('resistor', 'capacitor', 'inductor');

-- +goose Down
-- Nothing to restore; canonical rows are re-populated by SyncCanonicalAttributeDefinitions().
