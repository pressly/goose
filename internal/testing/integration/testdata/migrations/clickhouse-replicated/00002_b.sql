-- +goose Up
CREATE TABLE IF NOT EXISTS clickstream ON CLUSTER '{cluster}' (
    customer_id String,
    time_stamp Date,
    click_event_type String,
    country_code FixedString(2),
    source_id UInt64
)
ENGINE = ReplicatedMergeTree()
ORDER BY (time_stamp);

-- +goose Down
DROP TABLE IF EXISTS clickstream ON CLUSTER '{cluster}' SYNC;
