-- +goose Up
-- +goose StatementBegin
INSERT INTO owners(owner_id, owner_name, owner_type) 
    VALUES (1, 'lucas', 'user'), (2, 'spacey', 'organization');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM owners WHERE owner_id IN (1, 2);
-- +goose StatementEnd
