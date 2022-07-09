package integration

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v4"
	"github.com/pressly/goose/v4/internal/check"
	_ "modernc.org/sqlite"
)

func TestCollectMigrations(t *testing.T) {
	t.Parallel()

	t.Run("non-existent", func(t *testing.T) {
		dir := filepath.Join("testdata", "empty")
		_, err := goose.NewProvider(goose.DialectSqlite, newSqliteMemory(t), dir, nil)
		check.IsError(t, err, goose.ErrNoMigrations)
	})
	t.Run("duplicate-sql", func(t *testing.T) {
		dir := filepath.Join("testdata", "duplicate-sql")
		_, err := goose.NewProvider(goose.DialectSqlite, newSqliteMemory(t), dir, nil)
		check.IsError(t, err, goose.ErrDuplicateMigrations)
	})
}

func newSqliteMemory(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	check.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	err = db.Ping()
	check.NoError(t, err)
	return db
}
