-- +goose Up
-- +goose StatementBegin
INSERT INTO owners(owner_id, owner_name, owner_type)
VALUES (1, 'lucas', 'user'), (2, 'space', 'organization');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM owners;
-- +goose StatementEnd
