-- +goose Up
-- +goose StatementBegin
INSERT INTO testing.test_migrations_1 (version_id, is_applied) VALUES (1, true);
-- +goose StatementEnd
-- +goose StatementBegin
INSERT INTO testing.test_migrations_1 (version_id, is_applied) VALUES (2, true);
-- +goose StatementEnd
-- +goose StatementBegin
INSERT INTO testing.test_migrations_1 (version_id, is_applied) VALUES (3, true);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM testing.test_migrations_1 WHERE version_id < 10;
-- +goose StatementEnd
