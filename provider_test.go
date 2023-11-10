package goose_test

import (
	"database/sql"
	"errors"
	"io/fs"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
	_ "modernc.org/sqlite"
)

func TestProvider(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "sql_embed.db"))
	check.NoError(t, err)
	t.Run("empty", func(t *testing.T) {
		_, err := goose.NewProvider(goose.DialectSQLite3, db, fstest.MapFS{})
		check.HasError(t, err)
		check.Bool(t, errors.Is(err, goose.ErrNoMigrations), true)
	})

	mapFS := fstest.MapFS{
		"migrations/001_foo.sql": {Data: []byte(`-- +goose Up`)},
		"migrations/002_bar.sql": {Data: []byte(`-- +goose Up`)},
	}
	fsys, err := fs.Sub(mapFS, "migrations")
	check.NoError(t, err)
	p, err := goose.NewProvider(goose.DialectSQLite3, db, fsys)
	check.NoError(t, err)
	sources := p.ListSources()
	check.Equal(t, len(sources), 2)
	check.Equal(t, sources[0], newSource(goose.TypeSQL, "001_foo.sql", 1))
	check.Equal(t, sources[1], newSource(goose.TypeSQL, "002_bar.sql", 2))

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
