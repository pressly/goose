package integration

import (
	"testing"

	"github.com/pressly/goose/v3/database"
	"github.com/pressly/goose/v3/internal/testing/integration/testdb"
	"github.com/stretchr/testify/require"
)

func TestPostgres(t *testing.T) {
	t.Parallel()

	db, cleanup, err := testdb.NewPostgres()
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.NoError(t, db.Ping())

	testDatabase(t, database.DialectPostgres, db, "testdata/migrations/postgres")
}

func TestClickhouse(t *testing.T) {
	t.Parallel()

	db, cleanup, err := testdb.NewClickHouse()
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.NoError(t, db.Ping())

	testDatabase(t, database.DialectClickHouse, db, "testdata/migrations/clickhouse")
}

func TestMySQL(t *testing.T) {
	t.Parallel()

	db, cleanup, err := testdb.NewMariaDB()
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.NoError(t, db.Ping())

	testDatabase(t, database.DialectMySQL, db, "testdata/migrations/mysql")
}

func TestTurso(t *testing.T) {
	t.Parallel()

	db, cleanup, err := testdb.NewTurso()
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.NoError(t, db.Ping())

	testDatabase(t, database.DialectTurso, db, "testdata/migrations/turso")
}
