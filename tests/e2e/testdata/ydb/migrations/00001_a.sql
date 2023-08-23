-- +goose Up
-- +goose StatementBegin
CREATE TABLE owners (
    owner_id Uint64,
    owner_name Utf8,
    owner_type Utf8,
    PRIMARY KEY (owner_id)
);
CREATE TABLE repos (
    repo_id Uint64,
    repo_owner_id Uint64,
    repo_full_name Utf8,
    PRIMARY KEY (repo_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE repos;
DROP TABLE owners;
-- +goose StatementEnd
