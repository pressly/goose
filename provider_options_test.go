package goose_test

import (
	"database/sql"
	"io/fs"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
)

func TestNewProvider(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "sql_embed.db"))
	check.NoError(t, err)
	fsys := newFsys()
	t.Run("invalid", func(t *testing.T) {
		// Empty dialect not allowed
		_, err = goose.NewProvider("", db, fsys)
		check.HasError(t, err)
		// Invalid dialect not allowed
		_, err = goose.NewProvider("unknown-dialect", db, fsys)
		check.HasError(t, err)
		// Nil db not allowed
		_, err = goose.NewProvider("sqlite3", nil, fsys)
		check.HasError(t, err)
		// Nil fsys not allowed
		_, err = goose.NewProvider("sqlite3", db, nil)
		check.HasError(t, err)
		// Duplicate table name not allowed
		_, err = goose.NewProvider("sqlite3", db, fsys, goose.WithTableName("foo"), goose.WithTableName("bar"))
		check.HasError(t, err)
		check.Equal(t, `table already set to "foo"`, err.Error())
		// Empty table name not allowed
		_, err = goose.NewProvider("sqlite3", db, fsys, goose.WithTableName(""))
		check.HasError(t, err)
		check.Equal(t, "table must not be empty", err.Error())
	})
	t.Run("valid", func(t *testing.T) {
		// Valid dialect, db, and fsys allowed
		_, err = goose.NewProvider("sqlite3", db, fsys)
		check.NoError(t, err)
		// Valid dialect, db, fsys, and table name allowed
		_, err = goose.NewProvider("sqlite3", db, fsys, goose.WithTableName("foo"))
		check.NoError(t, err)
		// Valid dialect, db, fsys, and verbose allowed
		_, err = goose.NewProvider("sqlite3", db, fsys, goose.WithVerbose())
		check.NoError(t, err)
	})
}

func newFsys() fs.FS {
	return fstest.MapFS{
		"1_foo.sql": {Data: []byte(migration1)},
		"2_bar.sql": {Data: []byte(migration2)},
		"3_baz.sql": {Data: []byte(migration3)},
		"4_qux.sql": {Data: []byte(migration4)},
	}
}

var (
	migration1 = `
-- +goose Up
CREATE TABLE foo (id INTEGER PRIMARY KEY);
-- +goose Down
DROP TABLE foo;
`
	migration2 = `
-- +goose Up
ALTER TABLE foo ADD COLUMN name TEXT;
-- +goose Down
ALTER TABLE foo DROP COLUMN name;
`
	migration3 = `
-- +goose Up
CREATE TABLE bar (
    id INTEGER PRIMARY KEY,
    description TEXT
);
-- +goose Down
DROP TABLE bar;
`
	migration4 = `
-- +goose Up
-- Rename the 'foo' table to 'my_foo'
ALTER TABLE foo RENAME TO my_foo;

-- Add a new column 'timestamp' to 'my_foo'
ALTER TABLE my_foo ADD COLUMN timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- +goose Down
-- Remove the 'timestamp' column from 'my_foo'
ALTER TABLE my_foo DROP COLUMN timestamp;

-- Rename the 'my_foo' table back to 'foo'
ALTER TABLE my_foo RENAME TO foo;
`
)
