-- +goose Up
ALTER TABLE owners ADD owner_email NVARCHAR(255) NULL;

-- +goose Down
ALTER TABLE owners DROP COLUMN owner_email;
