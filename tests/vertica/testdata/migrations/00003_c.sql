-- +goose Up
-- +goose StatementBegin
INSERT INTO testing.dim_test_scd VALUES (1, '575a0dd4-bd97-44ac-aae0-987090181da8', '2021-10-02', '2021-10-03', false, '123');
-- +goose StatementEnd
-- +goose StatementBegin
INSERT INTO testing.dim_test_scd VALUES (2, '575a0dd4-bd97-44ac-aae0-987090181da8', '2021-10-03', '2021-10-04', false, '456');
-- +goose StatementEnd
-- +goose StatementBegin
INSERT INTO testing.dim_test_scd VALUES (3, '575a0dd4-bd97-44ac-aae0-987090181da8', '2021-10-04', '9999-12-31', true, '789');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM testing.dim_test_scd where test_id = '575a0dd4-bd97-44ac-aae0-987090181da8';
-- +goose StatementEnd
