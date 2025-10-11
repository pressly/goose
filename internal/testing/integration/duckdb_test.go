//go:build duckdb
// +build duckdb

package integration

import (
	"testing"

	"github.com/pressly/goose/v3/database"
	"github.com/pressly/goose/v3/internal/testing/testdb"
	"github.com/stretchr/testify/require"
)

func TestDuckDB(t *testing.T) {
	t.Parallel()

	db, cleanup, err := testdb.NewDuckDB()
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.NoError(t, db.Ping())

	testDatabase(t, database.DialectDuckDB, db, "testdata/migrations/duckdb")
}
