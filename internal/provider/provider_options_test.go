package provider_test

import (
	"database/sql"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3/internal/check"
	"github.com/pressly/goose/v3/internal/provider"
	"github.com/pressly/goose/v3/state/storage"
	_ "modernc.org/sqlite"
)

func TestNewProvider(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "sql_embed.db"))
	check.NoError(t, err)
	fsys := fstest.MapFS{
		"1_foo.sql": {Data: []byte(migration1)},
		"2_bar.sql": {Data: []byte(migration2)},
		"3_baz.sql": {Data: []byte(migration3)},
		"4_qux.sql": {Data: []byte(migration4)},
	}
	t.Run("invalid", func(t *testing.T) {
		// Empty dialect not allowed
		_, err = provider.NewProvider(nil, db, fsys)
		check.HasError(t, err)
		// Nil db not allowed
		_, err = provider.NewProvider(storage.Sqlite3(), nil, fsys)
		check.HasError(t, err)
		// Nil fsys not allowed
		_, err = provider.NewProvider(storage.Sqlite3(), db, nil)
		check.HasError(t, err)
	})
	t.Run("valid", func(t *testing.T) {
		// Valid dialect, db, and fsys allowed
		_, err = provider.NewProvider(storage.Sqlite3(), db, fsys)
		check.NoError(t, err)
		// Valid dialect, db, fsys, and table name allowed
		_, err = provider.NewProvider(storage.Sqlite3WithTableName("foo"), db, fsys)
		check.NoError(t, err)
		// Valid dialect, db, fsys, and verbose allowed
		_, err = provider.NewProvider(storage.Sqlite3(), db, fsys,
			provider.WithVerbose(testing.Verbose()),
		)
		check.NoError(t, err)
	})
}
