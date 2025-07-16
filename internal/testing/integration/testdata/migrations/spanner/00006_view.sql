-- +goose NO TRANSACTION

-- +goose Up
-- +goose StatementBegin
CREATE VIEW view_owners
SQL SECURITY INVOKER AS
SELECT
  owners.owner_id,
  owners.owner_name,
  owners.owner_type
FROM owners
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW view_owners
-- +goose StatementEnd
