-- +goose Up

-- Insert 150 more owners.
INSERT INTO owners (owner_name)
WITH numbers AS (
    SELECT 101 AS n
    UNION ALL
    SELECT n + 1 FROM numbers WHERE n < 250
)
SELECT 'seed-user-' || n FROM numbers;

-- +goose Down

-- NOTE: there are 4 migration owners and 100 seed owners, that's why owner_id starts at 105
DELETE FROM owners WHERE owner_name LIKE 'seed-user-%' AND owner_id BETWEEN 105 AND 254;
