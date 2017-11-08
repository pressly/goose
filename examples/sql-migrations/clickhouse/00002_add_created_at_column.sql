-- +goose Up
ALTER TABLE events ADD COLUMN CreatedAt DateTime default now(); 
-- +goose Down
ALTER TABLE events DROP COLUMN CreatedAt;