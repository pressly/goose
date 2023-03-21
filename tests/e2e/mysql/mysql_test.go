package mysql_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v4"
	"github.com/pressly/goose/v4/internal/check"
	"github.com/pressly/goose/v4/internal/testdb"
)

func TestUpDownAll(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	migrationDir := filepath.Join("testdata", "migrations")
	te := newTestEnv(t, migrationDir)
	migrations := te.provider.ListMigrations()
	check.Number(t, len(migrations), 8)

	currentVersion, err := te.provider.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, currentVersion, 0)

	upResult, err := te.provider.Up(ctx)
	check.NoError(t, err)
	check.Number(t, len(upResult), 8)

	_, err = te.provider.DownTo(ctx, 0)
	check.NoError(t, err)

	currentVersion, err = te.provider.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, currentVersion, 0)
}

type te struct {
	provider *goose.Provider
	db       *sql.DB
}

func newTestEnv(t *testing.T, dir string) *te {
	t.Helper()

	db, cleanup, err := testdb.NewMariaDB()
	check.NoError(t, err)
	t.Cleanup(cleanup)
	options := goose.DefaultOptions().
		SetVerbose(testing.Verbose()).
		SetDir(dir)
	provider, err := goose.NewProvider(goose.DialectMySQL, db, options)
	check.NoError(t, err)
	check.NoError(t, provider.Ping(context.Background()))
	t.Cleanup(func() {
		check.NoError(t, provider.Close())
	})
	return &te{
		provider: provider,
		db:       db,
	}
}
