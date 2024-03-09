-- +goose NO TRANSACTION

-- +goose Up
CREATE UNIQUE INDEX owners_idx ON owners (owner_name);

-- +goose Down
-- +goose StatementBegin
DROP INDEX owners_idx;
-- +goose StatementEnd
