package gomigrations

import (
	"context"
	"database/sql"
	"testing"

	"github.com/pressly/goose/v4"
	"github.com/pressly/goose/v4/internal/check"
	"github.com/pressly/goose/v4/internal/testdb"

	_ "github.com/pressly/goose/v4/tests/gomigrations/error/testdata"
)

func TestGoMigrationByOne(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	db, cleanup, err := testdb.NewPostgres()
	check.NoError(t, err)
	t.Cleanup(cleanup)
	{
		options := goose.DefaultOptions().
			SetDir("testdata").
			SetVerbose(testing.Verbose())
		p, err := goose.NewProvider(goose.DialectPostgres, db, options)
		check.NoError(t, err)
		check.Number(t, len(p.ListMigrations()), 4)

		currentVersion, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, 0)

		_, err = p.UpByOne(ctx)
		check.NoError(t, err)
		currentVersion, err = p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, 1)

		// Registered Go migration run outside a goose tx using *sql.DB.
		_, err = p.UpByOne(ctx)
		check.HasError(t, err)
		// Error from Go migration:
		// 		failed to run Go migration: 002_ERROR_insert_no_tx.go: simulate error: too many inserts
		check.Contains(t, err.Error(), "failed to run Go migration")
		check.Contains(t, err.Error(), "simulate error: too many inserts")
		currentVersion, err = p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, 1)

		// This migration was inserting 100 rows, but fails at 50, and
		// because it's run outside a goose tx then we expect 50 rows.
		check.Number(t, countTableFoo(t, db), 50)
	}
	{
		options := goose.DefaultOptions().
			SetDir("testdata").
			SetVerbose(testing.Verbose()).
			SetExcludeVersions([]int64{2})
		p, err := goose.NewProvider(goose.DialectPostgres, db, options)
		check.NoError(t, err)
		check.Number(t, len(p.ListMigrations()), 3)

		// Truncate table so we have 0 rows.
		_, err = p.UpByOne(ctx)
		check.NoError(t, err)
		currentVersion, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		// We're at version 3, but keep in mind 2 was never applied because it failed.
		check.Number(t, currentVersion, 3)

		// Registered Go migration run within a tx.
		_, err = p.UpByOne(ctx)
		check.HasError(t, err)
		// Error from Go migration:
		// 		failed to run Go migration: 004_ERROR_insert.go: simulate error: too many inserts
		check.Contains(t, err.Error(), "failed to run Go migration")
		check.Contains(t, err.Error(), "simulate error: too many inserts")
		currentVersion, err = p.GetDBVersion(ctx)
		check.NoError(t, err)
		// This migration failed, so we're still at 3.
		check.Number(t, currentVersion, 3)
		// This migration was inserting 100 rows, but fails at 50. However, since it's
		// running within a tx we expect none of the inserts to persist.
		check.Number(t, countTableFoo(t, db), 0)
	}
}

func countTableFoo(t *testing.T, db *sql.DB) int {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM foo").Scan(&count)
	check.NoError(t, err)
	return count
}
