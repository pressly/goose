-- +goose Up
-- +goose StatementBegin
CREATE INDEX CONCURRENTLY concurrent_users_user_id_key ON users (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX CONCURRENTLY concurrent_users_user_id_key;
-- +goose StatementEnd
