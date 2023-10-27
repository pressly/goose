package provider_test

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3/database"
	"github.com/pressly/goose/v3/internal/check"
	"github.com/pressly/goose/v3/internal/provider"
	_ "modernc.org/sqlite"
)

func TestProvider(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "sql_embed.db"))
	check.NoError(t, err)
	t.Run("empty", func(t *testing.T) {
		_, err := provider.NewProvider("sqlite3", db, fstest.MapFS{})
		check.HasError(t, err)
		check.Bool(t, errors.Is(err, provider.ErrNoMigrations), true)
	})

	mapFS := fstest.MapFS{
		"migrations/001_foo.sql": {Data: []byte(`-- +goose Up`)},
		"migrations/002_bar.sql": {Data: []byte(`-- +goose Up`)},
	}
	fsys, err := fs.Sub(mapFS, "migrations")
	check.NoError(t, err)
	p, err := provider.NewProvider("sqlite3", db, fsys)
	check.NoError(t, err)
	sources := p.ListSources()
	check.Equal(t, len(sources), 2)
	check.Equal(t, sources[0], provider.NewSource(provider.TypeSQL, "001_foo.sql", 1))
	check.Equal(t, sources[1], provider.NewSource(provider.TypeSQL, "002_bar.sql", 2))

	t.Run("duplicate_go", func(t *testing.T) {
		// Not parallel because it modifies global state.
		register := []*provider.Migration{
			{
				Version: 1, Source: "00001_users_table.go", Registered: true,
				UpFnContext:   nil,
				DownFnContext: nil,
			},
		}
		err := provider.SetGlobalGoMigrations(register)
		check.NoError(t, err)
		t.Cleanup(provider.ResetGlobalGoMigrations)

		db := newDB(t)
		_, err = provider.NewProvider(database.DialectSQLite3, db, nil,
			provider.WithGoMigration(1, nil, nil),
		)
		check.HasError(t, err)
		check.Equal(t, err.Error(), "go migration with version 1 already registered")
	})
	t.Run("empty_go", func(t *testing.T) {
		db := newDB(t)
		// explicit
		_, err := provider.NewProvider(database.DialectSQLite3, db, nil,
			provider.WithGoMigration(1, &provider.GoMigration{Run: nil}, &provider.GoMigration{Run: nil}),
		)
		check.HasError(t, err)
		check.Contains(t, err.Error(), "go migration with version 1 must have an up function")
	})
	t.Run("duplicate_up", func(t *testing.T) {
		err := provider.SetGlobalGoMigrations([]*provider.Migration{
			{
				Version: 1, Source: "00001_users_table.go", Registered: true,
				UpFnContext:     func(context.Context, *sql.Tx) error { return nil },
				UpFnNoTxContext: func(ctx context.Context, db *sql.DB) error { return nil },
			},
		})
		t.Cleanup(provider.ResetGlobalGoMigrations)
		check.HasError(t, err)
		check.Contains(t, err.Error(), "must specify exactly one of UpFnContext or UpFnNoTxContext")
	})
	t.Run("duplicate_down", func(t *testing.T) {
		err := provider.SetGlobalGoMigrations([]*provider.Migration{
			{
				Version: 1, Source: "00001_users_table.go", Registered: true,
				DownFnContext:     func(context.Context, *sql.Tx) error { return nil },
				DownFnNoTxContext: func(ctx context.Context, db *sql.DB) error { return nil },
			},
		})
		t.Cleanup(provider.ResetGlobalGoMigrations)
		check.HasError(t, err)
		check.Contains(t, err.Error(), "must specify exactly one of DownFnContext or DownFnNoTxContext")
	})
	t.Run("not_registered", func(t *testing.T) {
		err := provider.SetGlobalGoMigrations([]*provider.Migration{
			{
				Version: 1, Source: "00001_users_table.go",
			},
		})
		t.Cleanup(provider.ResetGlobalGoMigrations)
		check.HasError(t, err)
		check.Contains(t, err.Error(), "migration must be registered")
	})
	t.Run("zero_not_allowed", func(t *testing.T) {
		err := provider.SetGlobalGoMigrations([]*provider.Migration{
			{
				Version: 0,
			},
		})
		t.Cleanup(provider.ResetGlobalGoMigrations)
		check.HasError(t, err)
		check.Contains(t, err.Error(), "migration versions must be greater than zero")
	})
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
