package goose_test

import (
	"database/sql"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

//go:embed testdata/migrations/*.sql
var embedMigrations embed.FS

func TestEmbeddedMigrations(t *testing.T) {
	dir := t.TempDir()
	// not using t.Parallel here to avoid races
	db, err := sql.Open("sqlite", filepath.Join(dir, "sql_embed.db"))
	require.NoError(t, err)

	db.SetMaxOpenConns(1)

	migrationFiles, err := fs.ReadDir(embedMigrations, "testdata/migrations")
	require.NoError(t, err)
	total := len(migrationFiles)

	// decouple from existing structure
	fsys, err := fs.Sub(embedMigrations, "testdata/migrations")
	require.NoError(t, err)

	goose.SetBaseFS(fsys)
	t.Cleanup(func() { goose.SetBaseFS(nil) })
	require.NoError(t, goose.SetDialect("sqlite3"))

	t.Run("migration_cycle", func(t *testing.T) {
		err := goose.Up(db, ".")
		require.NoError(t, err)
		ver, err := goose.GetDBVersion(db)
		require.NoError(t, err)
		require.EqualValues(t, ver, total)
		err = goose.Reset(db, ".")
		require.NoError(t, err)
		ver, err = goose.GetDBVersion(db)
		require.NoError(t, err)
		require.EqualValues(t, 0, ver)
	})
	t.Run("create_uses_os_fs", func(t *testing.T) {
		dir := t.TempDir()
		err := goose.Create(db, dir, "test", "sql")
		require.NoError(t, err)
		paths, _ := filepath.Glob(filepath.Join(dir, "*test.sql"))
		require.NotEmpty(t, paths)
		err = goose.Fix(dir)
		require.NoError(t, err)
		_, err = os.Stat(filepath.Join(dir, "00001_test.sql"))
		require.NoError(t, err)
	})
}
