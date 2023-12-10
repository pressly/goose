//go:build !go1.21
// +build !go1.21

package testdb

import "database/sql"

// NewTurso is a stub for older go versions.
func NewTurso(options ...OptionsFunc) (db *sql.DB, cleanup func(), err error) {
	panic("not implemented for < go 1.21")
}
