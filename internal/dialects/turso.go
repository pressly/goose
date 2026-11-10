package dialects

import "github.com/pressly/goose/v3/database/dialect"

// NewTurso returns a [dialect.QuerierExtender] for Turso dialect.
func NewTurso() dialect.QuerierExtender {
	return &turso{}
}

type turso struct {
	sqlite3
}

var _ dialect.QuerierExtender = (*turso)(nil)
