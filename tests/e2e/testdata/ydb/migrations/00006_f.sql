-- +goose Up
--ydb:SCHEME
CREATE TABLE IF NOT EXISTS stargazers (
    stargazer_repo_id Uint64,
    stargazer_owner_id Uint64,
    stargazer_starred_at DateTime,
    stargazer_location Utf8
);

-- +goose Down
--ydb:SCHEME
DROP TABLE stargazers;
