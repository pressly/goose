-- +goose Up
-- +goose StatementBegin
ALTER TABLE repos
    ADD COLUMN homepage_url Utf8,
    ADD COLUMN is_private Bool;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE repos
    DROP COLUMN  homepage_url,
    DROP COLUMN is_private;
-- +goose StatementEnd
