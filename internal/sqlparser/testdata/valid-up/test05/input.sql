-- +goose Up

CREATE TABLE ssh_keys (
    id integer NOT NULL,
    "publicKey" text
-- insert comment there
);
-- insert comment there

-- This is a dangling comment
-- Another comment
-- Foo comment

CREATE TABLE ssh_keys_backup (
    id integer NOT NULL,
    -- insert comment here
    "publicKey" text
    -- insert comment there
);


-- +goose Down
