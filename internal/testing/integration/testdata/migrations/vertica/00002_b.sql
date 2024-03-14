-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS testing.dim_test_scd
(
    test_key    BIGINT  NOT NULL,
    test_id     UUID    NOT NULL,
    valid_from  DATE    NOT NULL,
    valid_to    DATE    NOT NULL,
    is_current  BOOLEAN NOT NULL
        DEFAULT (valid_to = '9999/12/31'),
    external_id VARCHAR(100)
) UNSEGMENTED ALL NODES;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE PROJECTION IF NOT EXISTS testing.dim_test_scd_proj_is_current AS
    SELECT test_key,
           test_id,
           valid_from,
           valid_to,
           is_current,
           external_id
    FROM testing.dim_test_scd
    ORDER BY is_current, test_id
    SEGMENTED BY HASH(test_key) ALL NODES;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE OR REPLACE VIEW testing.Test AS
SELECT test_key,
       test_id,
       external_id
FROM testing.dim_test_scd
WHERE is_current = true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS testing.Test;
-- +goose StatementEnd
-- +goose StatementBegin
DROP PROJECTION IF EXISTS testing.dim_test_scd_proj_is_current;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS testing.dim_test_scd;
-- +goose StatementEnd
