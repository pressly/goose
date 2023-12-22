-- +goose ENVSUB ON
-- +goose Up
CREATE TABLE ${GOOSE_ENV_REGION}post (
		id int NOT NULL,
		title text,
		body text,
		PRIMARY KEY(id)
);                  -- 1st stmt

-- comment
SELECT 2;           -- 2nd stmt
SELECT 3; SELECT 3; -- 3rd stmt
SELECT 4;           -- 4th stmt

-- +goose Down
-- comment
DROP TABLE ${GOOSE_ENV_REGION}post;    -- 1st stmt
