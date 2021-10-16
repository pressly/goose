-- +goose Up
-- +goose StatementBegin
-- This migration intentionally depends on 00006_f.sql
ALTER TABLE stargazers DROP COLUMN stargazer_location;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE stargazers ADD COLUMN stargazer_location text NOT NULL;
-- +goose StatementEnd
