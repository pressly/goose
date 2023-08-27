-- +goose NO TRANSACTION

-- +goose Up
CREATE VIEW posts_view AS
    SELECT
        p.id,
        p.title,
        p.content,
        p.created_at,
        u.username AS author
    FROM posts p
    JOIN users u ON p.author_id = u.id;

-- +goose Down
DROP VIEW posts_view;
