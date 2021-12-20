-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS stargazers (
    stargazer_repo_id Uint64,
    stargazer_owner_id Uint64,
    stargazer_starred_at DateTime,
    stargazer_location Utf8
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE stargazers;
-- +goose StatementEnd
