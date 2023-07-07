-- +goose Up
SELECT * FROM foo;
SELECT * FROM bar
-- +goose Down
SELECT * FROM baz;
