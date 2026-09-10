-- +goose Up
CREATE TABLE IF NOT EXISTS events ON CLUSTER goose_cluster (
    id UInt64,
    payload String,
    ts DateTime DEFAULT now()
)
ENGINE = ReplicatedMergeTree
ORDER BY (id);

-- +goose Down
DROP TABLE IF EXISTS events ON CLUSTER goose_cluster SYNC;
