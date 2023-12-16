-- +goose Up
-- +goose StatementBegin
CREATE TABLE stargazers (
    stargazer_repo_id integer,
    stargazer_owner_id integer,
    stargazer_starred_at integer,
    stargazer_location text,
    PRIMARY KEY (stargazer_repo_id, stargazer_owner_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE stargazers;
-- +goose StatementEnd
