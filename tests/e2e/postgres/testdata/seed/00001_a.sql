-- +goose Up
-- +goose StatementBegin

-- Insert 100 owners.
INSERT INTO owners (owner_name, owner_type)
SELECT
	'seed-user-' || i,
	(SELECT('{user,organization}'::owner_type []) [MOD(i, 2)+1])
FROM
	generate_series(1, 100) s (i);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- NOTE: there are 4 existing users from the migrations, that's why owner_id starts at 5
DELETE FROM owners where owner_name LIKE 'seed-user-%' AND owner_id BETWEEN 5 AND 104;
SELECT setval('owners_owner_id_seq', COALESCE((SELECT MAX(owner_id)+1 FROM owners), 1), false);
-- +goose StatementEnd