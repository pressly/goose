package testdb

import "database/sql"

// NewClickHouse starts a ClickHouse docker container. Returns db connection and a docker cleanup function.
func NewClickHouse(confDir string, options ...OptionsFunc) (db *sql.DB, cleanup func(), err error) {
	return newClickHouse(confDir, options...)
}

// NewPostgres starts a PostgreSQL docker container. Returns db connection and a docker cleanup function.
func NewPostgres(options ...OptionsFunc) (db *sql.DB, cleanup func(), err error) {
	return newPostgres(options...)
}

// NewMariaDB starts a MariaDB docker container. Returns a db connection and a docker cleanup function.
func NewMariaDB(options ...OptionsFunc) (db *sql.DB, cleanup func(), err error) {
	return newMariaDB(options...)
}

// NewVertica starts a Vertica docker container. Returns a db connection and a docker cleanup function.
func NewVertica(options ...OptionsFunc) (db *sql.DB, cleanup func(), err error) {
	return newVertica(options...)
}
