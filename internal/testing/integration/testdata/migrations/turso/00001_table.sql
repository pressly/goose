-- +goose Up
-- +goose StatementBegin
CREATE TABLE owners (
    owner_id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_name TEXT NOT NULL,
    owner_type TEXT CHECK(owner_type IN ('user', 'organization')) NOT NULL
);

CREATE TABLE IF NOT EXISTS repos (
    repo_id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_full_name TEXT NOT NULL,
    repo_owner_id INTEGER NOT NULL REFERENCES owners(owner_id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS repos;
DROP TABLE IF EXISTS owners;
-- +goose StatementEnd
