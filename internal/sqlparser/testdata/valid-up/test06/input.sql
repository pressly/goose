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

-- +goose StatementBegin




-- 1 this comment will NOT be preserved
  -- 2 this comment will NOT be preserved


CREATE FUNCTION do_something(sql TEXT) RETURNS INTEGER AS $$
BEGIN
  -- initiate technology
  PERFORM something_or_other(sql);

  -- increase technology
  PERFORM some_other_thing(sql);

  -- technology was successful
  RETURN 1;
END;
$$ LANGUAGE plpgsql;

-- 3 this comment WILL BE preserved
  -- 4 this comment WILL BE preserved


-- +goose StatementEnd

INSERT INTO post (id, title, body)
VALUES ('id_01', 'my_title', '
this is an insert statement including empty lines.

empty (blank) lines can be meaningful.

leave the lines to keep the text syntax.
');

-- +goose Down
TRUNCATE TABLE post; 
