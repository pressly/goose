-- +goose Up
-- +goose StatementBegin
-- CREATE FUNCTION placeholder, not needed yet.
-- +goose StatementEnd
CREATE TABLE users (
    id INTEGER PRIMARY KEY
);

-- +goose Down
DROP TABLE users;
