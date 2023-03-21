-- +goose NO TRANSACTION

-- +goose Up
DROP INDEX IF EXISTS owners_owner_name_idx;

-- +goose Down
CREATE UNIQUE INDEX CONCURRENTLY ON owners(owner_name);