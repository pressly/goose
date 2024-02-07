package testdb

import (
	"database/sql"
	"os"

	_ "github.com/marcboeker/go-duckdb"
)

func newDuckDB(opts ...OptionsFunc) (*sql.DB, func(), error) {
	option := &options{}
	for _, f := range opts {
		f(option)
	}

	db, err := sql.Open("duckdb", option.databaseFile)

	cleanup := func() {
		_ = db.Close()

		_ = os.Remove(option.databaseFile)
	}

	return db, cleanup, err
}
