-- +goose Up
INSERT INTO owners (owner_name, owner_type) VALUES ('linus', 'user');
INSERT INTO owners (owner_name, owner_type) VALUES ('torvalds', 'organization');
INSERT INTO repos (repo_full_name, repo_owner_id) VALUES ('linux', 1);
INSERT INTO repos (repo_full_name, repo_owner_id) VALUES ('linux-2.6', 1);

-- +goose Down
DELETE FROM repos;
DELETE FROM owners;
