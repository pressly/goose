-- +goose Up
-- +goose StatementBegin
INSERT INTO owners (owner_id, owner_name, owner_type) VALUES
  (1, 'lucas', 'user'),
  (2, 'space', 'organization'),
  (3, 'james', 'user'),
  (4, 'pressly', 'organization');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM owners WHERE owner_id IS NOT NULL
-- +goose StatementEnd
