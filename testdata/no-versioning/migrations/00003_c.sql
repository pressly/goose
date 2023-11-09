-- +goose Up
-- +goose StatementBegin
INSERT INTO owners(owner_name) VALUES ('james'), ('space');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM owners WHERE owner_name IN ('james', 'space');
-- +goose StatementEnd
