-- +goose NO TRANSACTION

-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE NULL_FILTERED INDEX owners_owner_name_idx ON owners (owner_name)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX owners_owner_name_idx
-- +goose StatementEnd
