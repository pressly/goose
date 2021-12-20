-- +goose Up
-- +goose StatementBegin
CREATE TABLE issues (
    issue_id Uint64,
    issue_created_by Uint64,
    issue_repo_id Uint64,
    issue_created_at DateTime,
    issue_description Utf8
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE issues;
-- +goose StatementEnd
