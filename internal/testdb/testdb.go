package testdb

import "database/sql"

// NewClickHouse starts a ClickHouse docker container, and returns
// a connection and a cleanup function.
// If bindPort is 0, a random port will be used.
func NewClickHouse(options ...OptionsFunc) (db *sql.DB, cleanup func(), err error) {
	return newClickHouse(options...)
}

// NewPostgres starts a PostgreSQL docker container, and returns
// a connection and a cleanup function.
// If bindPort is 0, a random port will be used.
func NewPostgres(options ...OptionsFunc) (db *sql.DB, cleanup func(), err error) {
	return newPostgres(options...)
}

// NewMariaDB starts a MariaDB docker container, and returns
// a connection and a cleanup function.
// If bindPort is 0, a random port will be used.
func NewMariaDB(options ...OptionsFunc) (db *sql.DB, cleanup func(), err error) {
	return newMariaDB(options...)
}
