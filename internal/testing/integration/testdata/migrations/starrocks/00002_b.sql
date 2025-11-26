-- +goose Up
-- +goose StatementBegin
CREATE TABLE testing.test_migrations_1 (
		version_id bigint NOT NULL,
		id bigint NOT NULL AUTO_INCREMENT,
		is_applied boolean NOT NULL,
		tstamp datetime NULL default CURRENT_TIMESTAMP
	)
	PRIMARY KEY (version_id,id)
	DISTRIBUTED BY HASH (id)
	ORDER BY (version_id);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TABLE testing.test_migrations_2 (
		version_id bigint NOT NULL,
		id bigint NOT NULL AUTO_INCREMENT,
		is_applied boolean NOT NULL,
		tstamp datetime NULL default CURRENT_TIMESTAMP
	)
	PRIMARY KEY (version_id,id)
	DISTRIBUTED BY HASH (id)
	ORDER BY (version_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS testing.test_migrations_1;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE IF EXISTS testing.test_migrations_2;
-- +goose StatementEnd
