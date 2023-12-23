CREATE TABLE post (
	id int NOT NULL,
	title text,
	$GOOSE_ENV_NAME text,
	${GOOSE_ENV_NAME}title3 text,
	${ANOTHER_VAR:-default}title4 text,
	${GOOSE_ENV_SET_BUT_EMPTY_VALUE-default}title5 text,
);