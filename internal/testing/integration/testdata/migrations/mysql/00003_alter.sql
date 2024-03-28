-- +goose Up
-- +goose StatementBegin
ALTER TABLE repos 
    ADD COLUMN IF NOT EXISTS homepage_url TEXT;

ALTER TABLE repos 
    ADD COLUMN is_private BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE repos
    DROP COLUMN IF EXISTS homepage_url;

ALTER TABLE repos
    DROP COLUMN IF EXISTS is_private;
-- +goose StatementEnd
