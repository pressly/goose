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

// Regression test for https://github.com/pressly/goose/issues/1165: an empty
// pre-existing version table means the database is at version zero, and must
// not error with ErrNoNextVersion.
func TestEnsureDBVersionEmptyTable(t *testing.T) {
	db := newEmptyVersionTableDB(t)

	version, err := goose.EnsureDBVersion(db)
	require.NoError(t, err)
	require.EqualValues(t, 0, version)
}

func TestUpToEmptyVersionTable(t *testing.T) {
	db := newEmptyVersionTableDB(t)

	dir := t.TempDir()
	migration := `-- +goose Up
CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT);
-- +goose Down
DROP TABLE users;
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "00001_users.sql"), []byte(migration), 0o600))

	require.NoError(t, goose.UpTo(db, dir, 1))

	version, err := goose.EnsureDBVersion(db)
	require.NoError(t, err)
	require.EqualValues(t, 1, version)
}

func newEmptyVersionTableDB(t *testing.T) *sql.DB {
	t.Helper()
	require.NoError(t, goose.SetDialect("sqlite3"))

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "goose.db"))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`CREATE TABLE goose_db_version (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		version_id INTEGER NOT NULL,
		is_applied INTEGER NOT NULL,
		tstamp TIMESTAMP DEFAULT (datetime('now'))
	)`)
	require.NoError(t, err)
	return db
}
