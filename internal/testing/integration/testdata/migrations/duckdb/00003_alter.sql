-- +goose Up
-- +goose StatementBegin
ALTER TABLE repos 
ADD COLUMN homepage_url TEXT;

ALTER TABLE repos 
ADD COLUMN is_private BOOLEAN DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE repos
DROP COLUMN homepage_url;

ALTER TABLE repos
DROP COLUMN is_private;
-- +goose StatementEnd
