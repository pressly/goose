-- +goose Up
CREATE TABLE party_users (
                       id int NOT NULL PRIMARY KEY,
                       username text,
                       name text,
                       surname text
);

INSERT INTO party_users VALUES
     (0, 'root', '', ''),
     (1, 'vojtechvitek', 'Vojtech', 'Vitek');

-- +goose Down
DROP TABLE party_users;