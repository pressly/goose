
-- +goose Up
-- +goose StatementBegin
INSERT INTO owners(owner_id, owner_name, owner_type)
VALUES (3, 'james', 'user'), (4, 'pressly', 'organization');
INSERT INTO repos(repo_id, repo_full_name, repo_owner_id)
VALUES (1, 'james/rover', 3), (2, 'pressly/goose', 4);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM owners WHERE (owner_id = 3 OR owner_id = 4);
DELETE FROM repos WHERE (repo_id = 1 OR repo_id = 2);
-- +goose StatementEnd
