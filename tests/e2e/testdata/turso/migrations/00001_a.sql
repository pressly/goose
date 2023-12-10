-- +goose Up
-- +goose StatementBegin
CREATE TABLE owners (
    owner_id integer,
    owner_name integer,
    owner_type integer,
    PRIMARY KEY (owner_id)
);
CREATE TABLE repos (
    repo_id integer,
    repo_owner_id integer,
    repo_full_name integer,
    PRIMARY KEY (repo_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE repos;
DROP TABLE owners;
-- +goose StatementEnd
