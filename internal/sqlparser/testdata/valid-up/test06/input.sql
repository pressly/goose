-- +goose Up

CREATE TABLE article (
    id text,
            content text);

INSERT INTO article (id, content) VALUES ('id_0001',  E'# My markdown doc

first paragraph

second paragraph');

INSERT INTO article (id, content) VALUES ('id_0002',  E'# My second 

markdown doc

first paragraph

-- with a comment
    -- with an indent comment

second paragraph');


-- +goose Down
