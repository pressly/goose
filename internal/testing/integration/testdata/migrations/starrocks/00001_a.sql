-- +goose Up
-- +goose StatementBegin
CREATE SCHEMA IF NOT EXISTS testing;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP SCHEMA IF EXISTS testing;
-- +goose StatementEnd
