-- +goose Up
-- +goose StatementBegin
--ydb:SCHEME
-- This migration intentionally depends on 00006_f.sql
ALTER TABLE stargazers DROP COLUMN stargazer_location;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
--ydb:SCHEME
ALTER TABLE stargazers ADD COLUMN stargazer_location Utf8;
-- +goose StatementEnd
