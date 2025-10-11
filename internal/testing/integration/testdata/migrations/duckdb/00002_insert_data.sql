-- +goose Up
INSERT INTO users (id, name, email) VALUES (1, 'Alice', 'alice@example.com');
INSERT INTO users (id, name, email) VALUES (2, 'Bob', 'bob@example.com');

-- +goose Down
DELETE FROM users WHERE id IN (1, 2);
