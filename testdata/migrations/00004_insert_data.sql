-- +goose Up
INSERT INTO users (id, username, email)
VALUES
    (1, 'john_doe', 'john@example.com'),
    (2, 'jane_smith', 'jane@example.com'),
    (3, 'alice_wonderland', 'alice@example.com');

INSERT INTO posts (id, title, content, author_id)
VALUES
    (1, 'Introduction to SQL', 'SQL is a powerful language for managing databases...', 1),
    (2, 'Data Modeling Techniques', 'Choosing the right data model is crucial...', 2),
    (3, 'Advanced Query Optimization', 'Optimizing queries can greatly improve...', 1);

INSERT INTO comments (id, post_id, user_id, content)
VALUES
    (1, 1, 3, 'Great introduction! Looking forward to more.'),
    (2, 1, 2, 'SQL can be a bit tricky at first, but practice helps.'),
    (3, 2, 1, 'You covered normalization really well in this post.');

-- +goose Down
DELETE FROM comments;
DELETE FROM posts;
DELETE FROM users;
