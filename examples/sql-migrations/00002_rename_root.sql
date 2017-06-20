-- +goose Up
UPDATE users SET username='admin' WHERE username='root';

UPDATE users SET username='admin' WHERE username='root';

UPDATE users SET username='admin' WHERE username='root';

-- +goose Down
-- +goose StatementBegin
UPDATE users SET username='root' WHERE username='admin';
-- +goose StatementEnd
