-- +goose Up
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name VARCHAR NOT NULL,
    email VARCHAR
);

-- +goose Down
DROP TABLE users;
