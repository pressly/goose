-- +goose Up
-- +goose StatementBegin
INSERT INTO owners(owner_name, owner_type) 
VALUES 
    ('lucas', 'user'),
    ('space', 'organization'),
    ('james', 'user'),
    ('pressly', 'organization');
-- +goose StatementEnd

INSERT INTO repos(repo_full_name, repo_owner_id) 
VALUES 
    ('james/rover', (SELECT owner_id FROM owners WHERE owner_name = 'james')),
    ('pressly/goose', (SELECT owner_id FROM owners WHERE owner_name = 'pressly'));

-- +goose Down
-- +goose StatementBegin
DELETE FROM owners;
-- +goose StatementEnd
