CREATE TABLE post (
	id int NOT NULL,
	title text,
	$NAME text,
	${NAME}title3 text,
	${ANOTHER_VAR:-default}title4 text,
	${SET_BUT_EMPTY_VALUE-default}title5 text,
);