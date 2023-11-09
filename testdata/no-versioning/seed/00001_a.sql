-- +goose Up
-- +goose StatementBegin
-- Insert 100 owners.
INSERT INTO owners (owner_name)
WITH numbers AS (
    SELECT 1 AS n
    UNION ALL
    SELECT n + 1 FROM numbers WHERE n < 100
)
SELECT 'seed-user-' || n FROM numbers;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Delete the previously inserted data.
DELETE FROM owners WHERE owner_name LIKE 'seed-user-%';
-- +goose StatementEnd
