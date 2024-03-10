package gomigrations

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
	_ "github.com/pressly/goose/v3/tests/gomigrations/error/testdata"
	_ "modernc.org/sqlite"
)

func TestGoMigrationByOne(t *testing.T) {
	tempDir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(tempDir, "test.db"))
	check.NoError(t, err)
	err = goose.SetDialect(string(goose.DialectSQLite3))
	check.NoError(t, err)
	// Create goose table.
	current, err := goose.EnsureDBVersion(db)
	check.NoError(t, err)
	check.Number(t, current, 0)
	// Collect migrations.
	dir := "testdata"
	migrations, err := goose.CollectMigrations(dir, 0, goose.MaxVersion)
	check.NoError(t, err)
	check.Number(t, len(migrations), 4)

	// Setup table.
	err = migrations[0].Up(db)
	check.NoError(t, err)
	version, err := goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, version, 1)

	// Registered Go migration run outside a goose tx using *sql.DB.
	err = migrations[1].Up(db)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "failed to run go migration")
	version, err = goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, version, 1)

	// This migration was inserting 100 rows, but fails at 50, and
	// because it's run outside a goose tx then we expect 50 rows.
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM foo").Scan(&count)
	check.NoError(t, err)
	check.Number(t, count, 50)

	// Truncate table so we have 0 rows.
	err = migrations[2].Up(db)
	check.NoError(t, err)
	version, err = goose.GetDBVersion(db)
	check.NoError(t, err)
	// We're at version 3, but keep in mind 2 was never applied because it failed.
	check.Number(t, version, 3)

	// Registered Go migration run within a tx.
	err = migrations[3].Up(db)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "failed to run go migration")
	version, err = goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, version, 3) // This migration failed, so we're still at 3.
	// This migration was inserting 100 rows, but fails at 50. However, since it's
	// running within a tx we expect none of the inserts to persist.
	err = db.QueryRow("SELECT COUNT(*) FROM foo").Scan(&count)
	check.NoError(t, err)
	check.Number(t, count, 0)

}
