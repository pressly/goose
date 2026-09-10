-- +goose Up
INSERT INTO events (id, payload, category) VALUES
    (1, 'first',  'a'),
    (2, 'second', 'b'),
    (3, 'third',  'c');

-- +goose Down
ALTER TABLE events ON CLUSTER goose_cluster DELETE WHERE id IN (1,2,3) SETTINGS mutations_sync=2;
