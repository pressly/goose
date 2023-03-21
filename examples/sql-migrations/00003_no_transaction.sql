-- +goose NO TRANSACTION
-- +goose Up
CREATE TABLE post (
    id int NOT NULL,
    title text,
    body text,
    PRIMARY KEY(id)
);

SELECT pg_sleep(30);

-- +goose Down
DROP TABLE post;
