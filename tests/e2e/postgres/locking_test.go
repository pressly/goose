package postgres_test

import (
	"context"
	"testing"

	"github.com/pressly/goose/v4"
	"github.com/pressly/goose/v4/internal/check"
	"golang.org/x/sync/errgroup"
)

func TestLockModeAdvisorySession(t *testing.T) {
	t.Parallel()

	// The migrations are written in such a way that they cannot be applied concurrently. This test
	// ensures that the advisory session lock mode works as expected.

	options := goose.DefaultOptions().
		SetDir(migrationsDir).
		SetVerbose(testing.Verbose()).
		SetLockMode(goose.LockModeAdvisorySession) // <----------------- advisory session lock mode

	te := newTestEnv(t, migrationsDir, &options)
	provider1 := te.provider

	provider2, err := goose.NewProvider(goose.DialectPostgres, te.db, options)
	check.NoError(t, err)

	migrations := provider1.ListMigrations()
	wantVersion := migrations[len(migrations)-1].Version

	var g errgroup.Group

	// Since the lock mode is advisory session, only one of these providers is expected to apply ALL
	// the migrations. The other provider should apply NO migrations. The test MUST fail if both
	// providers apply migrations.

	var res1, res2 int
	g.Go(func() error {
		ctx := context.Background()
		results, err := provider1.Up(ctx)
		check.NoError(t, err)
		res1 = len(results)
		currentVersion, err := provider1.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, wantVersion)
		return nil
	})
	g.Go(func() error {
		ctx := context.Background()
		results, err := provider2.Up(ctx)
		check.NoError(t, err)
		res2 = len(results)
		currentVersion, err := provider2.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, wantVersion)
		return nil
	})
	check.NoError(t, g.Wait())

	if res1 == 0 && res2 == 0 {
		t.Fatal("both providers applied no migrations")
	}
	if res1 > 0 && res2 > 0 {
		t.Fatal("both providers applied migrations")
	}
}
