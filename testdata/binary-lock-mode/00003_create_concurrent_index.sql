-- +goose Up
CREATE UNIQUE INDEX CONCURRENTLY users_user_name_key ON users (user_name);

-- +goose Down
-- +goose StatementBegin
DROP INDEX users_user_name_key;
-- +goose StatementEnd
