package testdb

import "database/sql"

// NewClickHouse starts a ClickHouse docker container, and returns
// a connection and a cleanup function.
// If bindPort is 0,b  a random port will be used.
func NewClickHouse(options ...OptionsFunc) (_ *sql.DB, cleanup func(), _ error) {
	return newClickHouse(options...)
}
