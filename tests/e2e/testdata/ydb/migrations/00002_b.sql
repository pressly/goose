-- +goose Up
INSERT INTO owners(owner_name, owner_type)
    VALUES ('lucas', 'user'), ('space', 'organization');

-- +goose Down
DELETE FROM owners;
