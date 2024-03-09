-- +goose Up
-- +goose StatementBegin
ALTER TABLE repos 
    ADD COLUMN IF NOT EXISTS homepage_url text,
    ADD COLUMN is_private boolean NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE repos
    DROP COLUMN IF EXISTS homepage_url,
    DROP COLUMN is_private;
-- +goose StatementEnd
