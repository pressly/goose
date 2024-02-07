-- +goose Up
-- +goose StatementBegin
CREATE SEQUENCE owner_id;
CREATE TABLE owners (
    owner_id INTEGER PRIMARY KEY DEFAULT NEXTVAL('owner_id'),
    owner_name VARCHAR,
    owner_type VARCHAR
);
CREATE SEQUENCE repo_id;
CREATE TABLE repos (
    repo_id INTEGER PRIMARY KEY DEFAULT NEXTVAL('repo_id'),
    repo_owner_id INTEGER,
    repo_full_name VARCHAR
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE repos;
DROP SEQUENCE repo_id;
DROP TABLE owners;
DROP SEQUENCE owner_id;
-- +goose StatementEnd
