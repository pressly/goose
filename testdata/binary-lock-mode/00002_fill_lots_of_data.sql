-- +goose Up
-- +goose StatementBegin
INSERT INTO users (user_id) SELECT generate_series(1, 1000000);
-- Acquire an exclusive lock on the table
ALTER TABLE users ADD CONSTRAINT users_user_id_key UNIQUE (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP CONSTRAINT users_user_id_key;
TRUNCATE users;
-- +goose StatementEnd
