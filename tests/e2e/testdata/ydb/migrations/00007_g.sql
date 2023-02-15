-- +goose NO TRANSACTION
-- +goose Up
--ydb:SCHEME
CREATE TABLE issues (
    issue_id Uint64,
    issue_created_by Uint64,
    issue_repo_id Uint64,
    issue_created_at DateTime,
    issue_description Utf8
);

-- +goose Down
--ydb:SCHEME
DROP TABLE issues;
