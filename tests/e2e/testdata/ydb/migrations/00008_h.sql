-- +goose NO TRANSACTION
-- +goose Up
--ydb:SCHEME
-- This migration intentionally depends on 00006_f.sql
ALTER TABLE stargazers DROP COLUMN stargazer_location;

-- +goose Down
--ydb:SCHEME
ALTER TABLE stargazers ADD COLUMN stargazer_location Utf8;
