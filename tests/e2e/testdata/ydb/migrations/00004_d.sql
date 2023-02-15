-- +goose NO TRANSACTION
-- +goose Up
--ydb:SCHEME
ALTER TABLE repos 
    ADD COLUMN homepage_url utf8,
    ADD COLUMN is_private bool;

-- +goose Down
--ydb:SCHEME
ALTER TABLE repos
    DROP COLUMN homepage_url,
    DROP COLUMN is_private;
