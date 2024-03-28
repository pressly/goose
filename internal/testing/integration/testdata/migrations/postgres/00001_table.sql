-- +goose Up
-- +goose StatementBegin
CREATE TYPE owner_type as ENUM('user', 'organization');

CREATE TABLE owners (
    owner_id BIGSERIAL PRIMARY KEY,
    owner_name text NOT NULL,
    owner_type owner_type NOT NULL
);

CREATE TABLE IF NOT EXISTS repos (
    repo_id BIGSERIAL NOT NULL,
    repo_full_name text NOT NULL,
    repo_owner_id bigint NOT NULL REFERENCES owners(owner_id) ON DELETE CASCADE,

    PRIMARY KEY (repo_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS repos;
DROP TABLE IF EXISTS owners;
DROP TYPE owner_type;
-- +goose StatementEnd
