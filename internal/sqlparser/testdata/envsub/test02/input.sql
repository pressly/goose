-- +goose Up

-- +goose ENVSUB ON
CREATE TABLE post (
	id int NOT NULL,
	title text,
	$GOOSE_ENV_NAME text,
	${GOOSE_ENV_NAME}title3 text,
	${ANOTHER_VAR:-default}title4 text,
	${GOOSE_ENV_SET_BUT_EMPTY_VALUE-default}title5 text,
);
-- +goose ENVSUB OFF

CREATE TABLE post (
	id int NOT NULL,
	title text,
	$GOOSE_ENV_NAME text,
	${GOOSE_ENV_NAME}title3 text,
	${ANOTHER_VAR:-default}title4 text,
	${GOOSE_ENV_SET_BUT_EMPTY_VALUE-default}title5 text,
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION test_func()
RETURNS void AS $$
-- +goose ENVSUB ON
BEGIN
	RAISE NOTICE '${GOOSE_ENV_NAME} \$GOOSE_ENV_NAME \$GOOSE_ENV_NAME';
END;
-- +goose ENVSUB OFF
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
