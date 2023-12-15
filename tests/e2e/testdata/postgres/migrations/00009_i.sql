-- +goose NO TRANSACTION

-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE INDEX CONCURRENTLY ON owners(owner_name);
-- +goose StatementEnd

-- +goose Down
DROP INDEX IF EXISTS owners_owner_name_idx;