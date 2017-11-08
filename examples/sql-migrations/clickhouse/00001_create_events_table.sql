-- +goose Up
CREATE TABLE IF NOT EXISTS events (
    EventDate Date,
    EventTime DateTime,
    OsID      UInt8
) Engine=MergeTree(EventDate, (EventDate), 8192);
-- +goose Down
DROP TABLE events;