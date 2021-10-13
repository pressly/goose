-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS stargazers (
    stargazer_repo_id bigint NOT NULL REFERENCES repos(repo_id) ON DELETE CASCADE,
    stargazer_owner_id bigint NOT NULL REFERENCES owners(owner_id) ON DELETE CASCADE,
    stargazer_starred_at timestamp NOT NULL,
    stargazer_location text NOT NULL
);

ALTER TABLE IF EXISTS stargazers
   ADD CONSTRAINT stargazers_repo_id_owner_id_key PRIMARY KEY (stargazer_repo_id, stargazer_owner_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE stargazers;
-- +goose StatementEnd
