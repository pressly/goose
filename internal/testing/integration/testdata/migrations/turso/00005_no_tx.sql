-- +goose NO TRANSACTION

-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS idx_owners_owner_name ON owners(owner_name);


-- +goose Down
DROP INDEX IF EXISTS idx_owners_owner_name;

