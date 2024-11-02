package gomigrations

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	_ "github.com/pressly/goose/v3/tests/gomigrations/error/testdata"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestGoMigrationByOne(t *testing.T) {
	tempDir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(tempDir, "test.db"))
	require.NoError(t, err)
	err = goose.SetDialect(string(goose.DialectSQLite3))
	require.NoError(t, err)
	// Create goose table.
	current, err := goose.EnsureDBVersion(db)
	require.NoError(t, err)
	require.EqualValues(t, 0, current)
	// Collect migrations.
	dir := "testdata"
	migrations, err := goose.CollectMigrations(dir, 0, goose.MaxVersion)
	require.NoError(t, err)
	require.Len(t, migrations, 4)

	// Setup table.
	err = migrations[0].Up(db)
	require.NoError(t, err)
	version, err := goose.GetDBVersion(db)
	require.NoError(t, err)
	require.EqualValues(t, 1, version)

	// Registered Go migration run outside a goose tx using *sql.DB.
	err = migrations[1].Up(db)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to run go migration")
	version, err = goose.GetDBVersion(db)
	require.NoError(t, err)
	require.EqualValues(t, 1, version)

	// This migration was inserting 100 rows, but fails at 50, and
	// because it's run outside a goose tx then we expect 50 rows.
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM foo").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 50, count)

	// Truncate table so we have 0 rows.
	err = migrations[2].Up(db)
	require.NoError(t, err)
	version, err = goose.GetDBVersion(db)
	require.NoError(t, err)
	// We're at version 3, but keep in mind 2 was never applied because it failed.
	require.EqualValues(t, 3, version)

	// Registered Go migration run within a tx.
	err = migrations[3].Up(db)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to run go migration")
	version, err = goose.GetDBVersion(db)
	require.NoError(t, err)
	require.EqualValues(t, 3, version) // This migration failed, so we're still at 3.
	// This migration was inserting 100 rows, but fails at 50. However, since it's
	// running within a tx we expect none of the inserts to persist.
	err = db.QueryRow("SELECT COUNT(*) FROM foo").Scan(&count)
	require.NoError(t, err)
	require.EqualValues(t, 0, count)

}
