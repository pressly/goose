package goose_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// TestEnsureDBVersionEmptyTable verifies that EnsureDBVersion returns version 0 without error when
// the version table exists but contains no rows (for example, when the table was pre-created
// manually).
func TestEnsureDBVersionEmptyTable(t *testing.T) {
	require.NoError(t, goose.SetDialect("sqlite3"))
	goose.SetTableName(goose.DefaultTablename)
	t.Cleanup(func() { goose.SetTableName(goose.DefaultTablename) })

	db := openSQLiteDB(t)
	createEmptyVersionTable(t, db)

	var count int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM goose_db_version`).Scan(&count))
	require.Zero(t, count, "version table must start empty")

	version, err := goose.EnsureDBVersion(db)
	require.NoError(t, err)
	require.Zero(t, version)

	// The table must remain empty — EnsureDBVersion must not insert any rows.
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM goose_db_version`).Scan(&count))
	require.Zero(t, count, "EnsureDBVersion must not insert rows into a pre-existing empty table")
}

// TestUpToEmptyVersionTable ensures UpTo applies migrations correctly when the version table
// already exists but has no rows.
func TestUpToEmptyVersionTable(t *testing.T) {
	require.NoError(t, goose.SetDialect("sqlite3"))
	goose.SetTableName(goose.DefaultTablename)
	t.Cleanup(func() { goose.SetTableName(goose.DefaultTablename) })

	db := openSQLiteDB(t)
	createEmptyVersionTable(t, db)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "00001_users.sql"), []byte(`-- +goose Up
CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);

-- +goose Down
DROP TABLE users;
`), 0o600))

	require.NoError(t, goose.UpTo(db, dir, 1))

	version, err := goose.GetDBVersion(db)
	require.NoError(t, err)
	require.EqualValues(t, 1, version)

	// A second run must be idempotent.
	require.NoError(t, goose.UpTo(db, dir, 1))
	version, err = goose.GetDBVersion(db)
	require.NoError(t, err)
	require.EqualValues(t, 1, version)
}

func openSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "goose.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

func createEmptyVersionTable(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`CREATE TABLE goose_db_version (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		version_id INTEGER NOT NULL,
		is_applied INTEGER NOT NULL,
		tstamp TIMESTAMP DEFAULT (datetime('now'))
	)`)
	require.NoError(t, err)
}
