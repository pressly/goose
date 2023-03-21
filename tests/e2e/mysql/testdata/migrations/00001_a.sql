-- +goose Up
-- +goose StatementBegin
CREATE TABLE owners (
    owner_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    owner_name varchar(255) NOT NULL,
    owner_type ENUM('user', 'organization') NOT NULL
);
CREATE TABLE repos (
    repo_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    repo_owner_id BIGINT NOT NULL,
    repo_full_name VARCHAR(255) NOT NULL,
    FOREIGN KEY (repo_owner_id) REFERENCES owners (owner_id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS repos;
DROP TABLE IF EXISTS owners;
-- +goose StatementEnd