-- +goose Up
SELECT * FROM bar
-- +goose Down
SELECT * FROM baz;
