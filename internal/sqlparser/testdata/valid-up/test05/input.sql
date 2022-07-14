-- +goose Up

CREATE TABLE ssh_keys (
    id integer NOT NULL,
    "publicKey" text
);

-- This is a dangling comment
-- Another comment
-- Foo comment

CREATE TABLE ssh_keys_backup (
    id integer NOT NULL,
    "publicKey" text
);


-- +goose Down
