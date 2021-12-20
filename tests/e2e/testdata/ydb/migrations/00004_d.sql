-- +goose Up
-- +goose StatementBegin
--ydb:SCHEME
ALTER TABLE repos 
    ADD COLUMN homepage_url utf8,
    ADD COLUMN is_private bool;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
--ydb:SCHEME
ALTER TABLE repos
    DROP COLUMN homepage_url,
    DROP COLUMN is_private;
-- +goose StatementEnd
