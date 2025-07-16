-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
CREATE TABLE owners (
    owner_id INT64 NOT NULL,
    owner_name STRING(255) NOT NULL,
    owner_type STRING(50) NOT NULL,
) PRIMARY KEY(owner_id)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE owners
-- +goose StatementEnd
