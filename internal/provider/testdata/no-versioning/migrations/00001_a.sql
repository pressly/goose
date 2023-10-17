-- +goose Up
CREATE TABLE owners (
    owner_id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_name TEXT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS owners;
