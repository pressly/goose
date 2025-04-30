package goose_test

import (
	"database/sql"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestNewProvider(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "sql_embed.db"))
	require.NoError(t, err)
	fsys := fstest.MapFS{
		"1_foo.sql": {Data: []byte(migration1)},
		"2_bar.sql": {Data: []byte(migration2)},
		"3_baz.sql": {Data: []byte(migration3)},
		"4_qux.sql": {Data: []byte(migration4)},
	}
	t.Run("invalid", func(t *testing.T) {
		// Empty dialect not allowed
		_, err = goose.NewProvider("", db, fsys)
		require.Error(t, err)
		// Invalid dialect not allowed
		_, err = goose.NewProvider("unknown-dialect", db, fsys)
		require.Error(t, err)
		// Nil db not allowed
		_, err = goose.NewProvider(goose.DialectSQLite3, nil, fsys)
		require.Error(t, err)
		// Nil store not allowed
		_, err = goose.NewProvider(goose.DialectSQLite3, db, nil, goose.WithStore(nil))
		require.Error(t, err)
		// Cannot set both dialect and store
		store, err := database.NewStore(goose.DialectSQLite3, "custom_table")
		require.NoError(t, err)
		_, err = goose.NewProvider(goose.DialectSQLite3, db, nil, goose.WithStore(store))
		require.Error(t, err)
		// Multiple stores not allowed
		_, err = goose.NewProvider(goose.DialectSQLite3, db, nil,
			goose.WithStore(store),
			goose.WithStore(store),
		)
		require.Error(t, err)
	})
	t.Run("valid", func(t *testing.T) {
		// Valid dialect, db, and fsys allowed
		_, err = goose.NewProvider(goose.DialectSQLite3, db, fsys)
		require.NoError(t, err)
		// Valid dialect, db, fsys, and verbose allowed
		_, err = goose.NewProvider(goose.DialectSQLite3, db, fsys,
			goose.WithVerbose(testing.Verbose()),
		)
		require.NoError(t, err)
		// Custom store allowed
		store, err := database.NewStore(goose.DialectSQLite3, "custom_table")
		require.NoError(t, err)
		_, err = goose.NewProvider("", db, nil, goose.WithStore(store))
		require.Error(t, err)
	})
}
