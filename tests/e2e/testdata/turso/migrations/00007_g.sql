-- +goose Up
-- +goose StatementBegin
CREATE TABLE issues (
    issue_id integer,
    issue_created_by integer,
    issue_repo_id integer,
    issue_created_at integer,
    issue_description text,
    PRIMARY KEY (issue_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE issues;
-- +goose StatementEnd
