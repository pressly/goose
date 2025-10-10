package dialects

import "github.com/pressly/goose/v3/database/dialect"

// NewDuckDB returns a [dialect.Querier] for DuckDB dialect.
//
// DuckDB is SQLite-compatible, so we embed the sqlite3 implementation.
func NewDuckDB() dialect.Querier {
	return &duckdb{}
}

type duckdb struct {
	sqlite3
}

var _ dialect.Querier = (*duckdb)(nil)
