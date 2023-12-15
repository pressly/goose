package goose_test

import (
	"database/sql"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
	_ "modernc.org/sqlite"
)

//go:embed testdata/migrations/*.sql
var embedMigrations embed.FS

func TestEmbeddedMigrations(t *testing.T) {
	dir := t.TempDir()
	// not using t.Parallel here to avoid races
	db, err := sql.Open("sqlite", filepath.Join(dir, "sql_embed.db"))
	check.NoError(t, err)

	db.SetMaxOpenConns(1)

	migrationFiles, err := fs.ReadDir(embedMigrations, "testdata/migrations")
	check.NoError(t, err)
	total := len(migrationFiles)

	// decouple from existing structure
	fsys, err := fs.Sub(embedMigrations, "testdata/migrations")
	check.NoError(t, err)

	goose.SetBaseFS(fsys)
	t.Cleanup(func() { goose.SetBaseFS(nil) })
	check.NoError(t, goose.SetDialect("sqlite3"))

	t.Run("migration_cycle", func(t *testing.T) {
		err := goose.Up(db, ".")
		check.NoError(t, err)
		ver, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, ver, total)
		err = goose.Reset(db, ".")
		check.NoError(t, err)
		ver, err = goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, ver, 0)
	})
	t.Run("create_uses_os_fs", func(t *testing.T) {
		dir := t.TempDir()
		err := goose.Create(db, dir, "test", "sql")
		check.NoError(t, err)
		paths, _ := filepath.Glob(filepath.Join(dir, "*test.sql"))
		check.NumberNotZero(t, len(paths))
		err = goose.Fix(dir)
		check.NoError(t, err)
		_, err = os.Stat(filepath.Join(dir, "00001_test.sql"))
		check.NoError(t, err)
	})
}
