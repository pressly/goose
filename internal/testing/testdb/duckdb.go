//go:build duckdb
// +build duckdb

package testdb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb"
)

// NewDuckDB creates a new DuckDB database for testing. Returns db connection and a cleanup function.
func NewDuckDB(opts ...OptionsFunc) (db *sql.DB, cleanup func(), err error) {
	option := &options{}
	for _, f := range opts {
		f(option)
	}

	// Create a temporary directory for the DuckDB database
	tmpDir, err := os.MkdirTemp("", "goose-duckdb-test-*")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")

	// Open DuckDB database
	db, err = sql.Open("duckdb", dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, nil, fmt.Errorf("failed to open DuckDB: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		os.RemoveAll(tmpDir)
		return nil, nil, fmt.Errorf("failed to ping DuckDB: %w", err)
	}

	cleanup = func() {
		db.Close()
		if !option.debug {
			os.RemoveAll(tmpDir)
		}
	}

	return db, cleanup, nil
}
