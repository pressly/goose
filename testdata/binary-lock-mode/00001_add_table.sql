-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
  user_id INTEGER
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users;
-- +goose StatementEnd
