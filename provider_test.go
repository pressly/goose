package goose_test

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestProvider(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "sql_embed.db"))
	require.NoError(t, err)
	t.Run("empty", func(t *testing.T) {
		_, err := goose.NewProvider(goose.DialectSQLite3, db, fstest.MapFS{})
		require.Error(t, err)
		require.ErrorIs(t, err, goose.ErrNoMigrations)
	})

	mapFS := fstest.MapFS{
		"migrations/001_foo.sql": {Data: []byte(`-- +goose Up`)},
		"migrations/002_bar.sql": {Data: []byte(`-- +goose Up`)},
	}
	fsys, err := fs.Sub(mapFS, "migrations")
	require.NoError(t, err)
	p, err := goose.NewProvider(goose.DialectSQLite3, db, fsys)
	require.NoError(t, err)
	sources := p.ListSources()
	require.Len(t, sources, 2)
	require.Equal(t, sources[0], newSource(goose.TypeSQL, "001_foo.sql", 1))
	require.Equal(t, sources[1], newSource(goose.TypeSQL, "002_bar.sql", 2))
}

func TestProviderSQLMigrationOrder(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name      string
		filenames [3]string
	}{
		{"unpadded", [3]string{"1_create.sql", "2_alter.sql", "10_insert.sql"}},
		{"padded", [3]string{"001_create.sql", "002_alter.sql", "010_insert.sql"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db := newDB(t)
			p, err := goose.NewProvider(goose.DialectSQLite3, db, fstest.MapFS{
				tt.filenames[0]: {Data: []byte(migration1)},
				tt.filenames[1]: {Data: []byte(migration2)},
				tt.filenames[2]: {Data: []byte("-- +goose Up\nINSERT INTO foo (id, name) VALUES (1, 'goose');")},
			}, goose.WithDisableGlobalRegistry(true))
			require.NoError(t, err)
			ctx := context.Background()
			t.Run("sources", func(t *testing.T) {
				sources := p.ListSources()
				require.Len(t, sources, 3)
				for i, version := range []int64{1, 2, 10} {
					require.Equal(t, newSource(goose.TypeSQL, tt.filenames[i], version), sources[i])
				}
			})
			t.Run("versions", func(t *testing.T) {
				current, target, err := p.GetVersions(ctx)
				require.NoError(t, err)
				require.Zero(t, current)
				require.EqualValues(t, 10, target)
			})
			t.Run("up", func(t *testing.T) {
				results, err := p.Up(ctx)
				require.NoError(t, err)
				require.Len(t, results, 3)
				var name string
				err = db.QueryRowContext(ctx, "SELECT name FROM foo WHERE id = 1").Scan(&name)
				require.NoError(t, err)
				require.Equal(t, "goose", name)
			})
		})
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

func TestPartialErrorUnwrap(t *testing.T) {
	err := &goose.PartialError{Err: goose.ErrNoCurrentVersion}
	require.ErrorIs(t, err, goose.ErrNoCurrentVersion)
}
