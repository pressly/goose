-- +goose Up
-- +goose StatementBegin
CREATE TABLE issues (
    issue_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    issue_created_by bigint NOT NULL REFERENCES owners(owner_id) ON DELETE CASCADE,
    issue_repo_id bigint NOT NULL REFERENCES repos(repo_id) ON DELETE CASCADE,
    issue_created_at timestamp NOT NULL,
    issue_description text NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE issues;
-- +goose StatementEnd
