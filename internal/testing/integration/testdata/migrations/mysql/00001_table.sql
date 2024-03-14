-- +goose Up
-- +goose StatementBegin
CREATE TABLE owners (
    owner_id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    owner_name VARCHAR(255) NOT NULL,
    owner_type ENUM('user', 'organization') NOT NULL
);

CREATE TABLE IF NOT EXISTS repos (
    repo_id BIGINT UNSIGNED AUTO_INCREMENT NOT NULL,
    repo_full_name VARCHAR(255) NOT NULL,
    repo_owner_id BIGINT UNSIGNED NOT NULL,
    
    PRIMARY KEY (repo_id),
    FOREIGN KEY (repo_owner_id) REFERENCES owners(owner_id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS repos;
DROP TABLE IF EXISTS owners;
-- +goose StatementEnd
