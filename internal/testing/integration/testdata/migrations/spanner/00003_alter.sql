-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
ALTER TABLE owners ADD COLUMN homepage_url STRING(255)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE owners DROP COLUMN homepage_url
-- +goose StatementEnd
