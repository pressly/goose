package testdb

import "database/sql"

// NewClickHouse starts a ClickHouse docker container, if successful,
// return a connection and a cleanup function.
func NewClickHouse() (_ *sql.DB, cleanup func(), _ error) {
	return newClickHouse()
}
