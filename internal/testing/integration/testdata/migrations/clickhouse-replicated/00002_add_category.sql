-- +goose Up
ALTER TABLE events ON CLUSTER goose_cluster
    ADD COLUMN IF NOT EXISTS category LowCardinality(String) DEFAULT '';

-- +goose Down
ALTER TABLE events ON CLUSTER goose_cluster
    DROP COLUMN IF EXISTS category;
