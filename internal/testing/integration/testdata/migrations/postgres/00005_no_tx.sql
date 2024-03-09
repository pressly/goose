-- +goose NO TRANSACTION

-- +goose Up
CREATE UNIQUE INDEX CONCURRENTLY ON owners(owner_name);

-- +goose Down
DROP INDEX IF EXISTS owners_owner_name_idx;
