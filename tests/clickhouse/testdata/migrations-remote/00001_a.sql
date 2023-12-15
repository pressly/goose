-- +goose Up
CREATE DICTIONARY taxi_zone_dictionary (
    LocationID UInt16 DEFAULT 0,
    Borough String,
    Zone String,
    service_zone String
)
PRIMARY KEY LocationID
SOURCE(HTTP(
    url 'https://datasets-documentation.s3.eu-west-3.amazonaws.com/nyc-taxi/taxi_zone_lookup.csv'
    format 'CSVWithNames'
))
LIFETIME(0)
LAYOUT(HASHED());

-- +goose Down
DROP DICTIONARY IF EXISTS taxi_zone_dictionary;
