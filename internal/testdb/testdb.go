package testdb

import (
	"github.com/pressly/goose/v3/internal"
)

// NewClickHouse starts a ClickHouse docker container. Returns db connection and a docker cleanup function.
func NewClickHouse(options ...OptionsFunc) (db internal.GooseDB, cleanup func(), err error) {
	conn, cleanFn, err := newClickHouse(options...)
	return internal.SqlToGooseAdapter{Conn: conn}, cleanFn, err
}

// NewPostgres starts a PostgreSQL docker container. Returns db connection and a docker cleanup function.
func NewPostgres(options ...OptionsFunc) (db internal.GooseDB, cleanup func(), err error) {
	return newPostgres(options...)
}

// NewMariaDB starts a MariaDB docker container. Returns a db connection and a docker cleanup function.
func NewMariaDB(options ...OptionsFunc) (db internal.GooseDB, cleanup func(), err error) {
	conn, cleanFn, err := newMariaDB(options...)
	return internal.SqlToGooseAdapter{Conn: conn}, cleanFn, err
}

// NewVertica starts a Vertica docker container. Returns a db connection and a docker cleanup function.
func NewVertica(options ...OptionsFunc) (db internal.GooseDB, cleanup func(), err error) {
	conn, cleanFn, err := newMariaDB(options...)
	return internal.SqlToGooseAdapter{Conn: conn}, cleanFn, err
}
