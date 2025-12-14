-- +goose Up
-- +goose StatementBegin
CREATE TABLE owners (
    owner_id BIGINT IDENTITY(1,1) PRIMARY KEY,
    owner_name NVARCHAR(255) NOT NULL,
    owner_type NVARCHAR(50) NOT NULL CHECK (owner_type IN ('user', 'organization'))
);

CREATE TABLE repos (
    repo_id BIGINT IDENTITY(1,1) NOT NULL,
    repo_full_name NVARCHAR(255) NOT NULL,
    repo_owner_id BIGINT NOT NULL,

    PRIMARY KEY (repo_id),
    CONSTRAINT FK_repos_owners FOREIGN KEY (repo_owner_id) REFERENCES owners(owner_id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS repos;
DROP TABLE IF EXISTS owners;
-- +goose StatementEnd
