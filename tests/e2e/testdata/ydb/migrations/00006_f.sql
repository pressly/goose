-- +goose Up
-- +goose StatementBegin
CREATE TABLE stargazers (
    stargazer_repo_id Uint64,
    stargazer_owner_id UInt64,
    stargazer_starred_at Timestamp,
    stargazer_location Utf8,
    PRIMARY KEY (stargazer_repo_id, stargazer_owner_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE stargazers;
-- +goose StatementEnd
