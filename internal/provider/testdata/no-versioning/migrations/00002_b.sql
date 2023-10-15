-- +goose Up
-- +goose StatementBegin
INSERT INTO owners(owner_name) VALUES ('lucas'), ('ocean');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM owners;
-- +goose StatementEnd
