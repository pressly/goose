-- +goose Up
-- +goose StatementBegin
ALTER TABLE repos
    ADD COLUMN homepage_url text;
ALTER TABLE repos
    ADD COLUMN is_private integer;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE repos
    DROP COLUMN  homepage_url;
ALTER TABLE repos
    DROP COLUMN is_private;
-- +goose StatementEnd
