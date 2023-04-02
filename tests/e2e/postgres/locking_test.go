package postgres_test

import (
	"context"
	"errors"
	"sort"
	"sync"
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
		SetLockMode(goose.LockModeAdvisorySession) /* ---------------- advisory session lock mode */

	te := newTestEnv(t, migrationsDir, &options)
	provider1 := te.provider

	provider2, err := goose.NewProvider(goose.DialectPostgres, te.db, options)
	check.NoError(t, err)

	migrations := provider1.ListMigrations()
	maxVersion := provider1.GetLastVersion()

	// Since the lock mode is advisory session, only one of these providers is expected to apply ALL
	// the migrations. The other provider should apply NO migrations. The test MUST fail if both
	// providers apply migrations.

	t.Run("up", func(t *testing.T) {
		var g errgroup.Group
		var res1, res2 int
		g.Go(func() error {
			ctx := context.Background()
			results, err := provider1.Up(ctx)
			check.NoError(t, err)
			res1 = len(results)
			currentVersion, err := provider1.GetDBVersion(ctx)
			check.NoError(t, err)
			check.Number(t, currentVersion, maxVersion)
			return nil
		})
		g.Go(func() error {
			ctx := context.Background()
			results, err := provider2.Up(ctx)
			check.NoError(t, err)
			res2 = len(results)
			currentVersion, err := provider2.GetDBVersion(ctx)
			check.NoError(t, err)
			check.Number(t, currentVersion, maxVersion)
			return nil
		})
		check.NoError(t, g.Wait())

		if res1 == 0 && res2 == 0 {
			t.Fatal("both providers applied no migrations")
		}
		if res1 > 0 && res2 > 0 {
			t.Fatal("both providers applied migrations")
		}
	})

	// Reset the database and run the same test with the advisory lock mode, but apply migrations
	// one-by-one.
	{
		_, err := provider1.Reset(context.Background())
		check.NoError(t, err)
		currentVersion, err := provider1.GetDBVersion(context.Background())
		check.NoError(t, err)
		check.Number(t, currentVersion, 0)
	}
	t.Run("up_by_one", func(t *testing.T) {
		var g errgroup.Group
		var (
			mu      sync.Mutex
			applied []int64
		)
		g.Go(func() error {
			for {
				result, err := provider1.UpByOne(context.Background())
				if err != nil {
					if errors.Is(err, goose.ErrNoNextVersion) {
						return nil
					}
					return err
				}
				check.NoError(t, err)
				mu.Lock()
				applied = append(applied, result.Migration.Version)
				mu.Unlock()
			}
		})
		g.Go(func() error {
			for {
				result, err := provider2.UpByOne(context.Background())
				if err != nil {
					if errors.Is(err, goose.ErrNoNextVersion) {
						return nil
					}
					return err
				}
				check.NoError(t, err)
				mu.Lock()
				applied = append(applied, result.Migration.Version)
				mu.Unlock()
			}
		})
		check.NoError(t, g.Wait())
		check.Number(t, len(applied), len(migrations))
		// sort.Slice(applied, func(i, j int) bool {
		//  return applied[i] < applied[j]
		// }) Each migration should have been applied up exactly once.
		for i := 0; i < len(migrations); i++ {
			check.Number(t, applied[i], migrations[i].Version)
		}
	})

	// Restore the database state by applying all migrations and run the same test with the advisory
	// lock mode, but apply down migrations in parallel.
	{
		_, err := provider1.Up(context.Background())
		check.NoError(t, err)
		currentVersion, err := provider1.GetDBVersion(context.Background())
		check.NoError(t, err)
		check.Number(t, currentVersion, maxVersion)
	}

	t.Run("down_to", func(t *testing.T) {
		var g errgroup.Group
		var res1, res2 int
		g.Go(func() error {
			ctx := context.Background()
			results, err := provider1.DownTo(ctx, 0)
			check.NoError(t, err)
			res1 = len(results)
			currentVersion, err := provider1.GetDBVersion(ctx)
			check.NoError(t, err)
			check.Number(t, currentVersion, 0)
			return nil
		})
		g.Go(func() error {
			ctx := context.Background()
			results, err := provider2.DownTo(ctx, 0)
			check.NoError(t, err)
			res2 = len(results)
			currentVersion, err := provider2.GetDBVersion(ctx)
			check.NoError(t, err)
			check.Number(t, currentVersion, 0)
			return nil
		})
		check.NoError(t, g.Wait())

		if res1 == 0 && res2 == 0 {
			t.Fatal("both providers applied no migrations")
		}
		if res1 > 0 && res2 > 0 {
			t.Fatal("both providers applied migrations")
		}
	})

	// Restore the database state by applying all migrations and run the same test with the advisory
	// lock mode, but apply down migrations one-by-one.
	{
		_, err := provider1.Up(context.Background())
		check.NoError(t, err)
		currentVersion, err := provider1.GetDBVersion(context.Background())
		check.NoError(t, err)
		check.Number(t, currentVersion, maxVersion)
	}

	t.Run("down_by_one", func(t *testing.T) {
		var g errgroup.Group
		var (
			mu      sync.Mutex
			applied []int64
		)
		g.Go(func() error {
			for {
				result, err := provider1.Down(context.Background())
				if err != nil {
					if errors.Is(err, goose.ErrNoCurrentVersion) {
						return nil
					}
					return err
				}
				check.NoError(t, err)
				mu.Lock()
				applied = append(applied, result.Migration.Version)
				mu.Unlock()
			}
		})
		g.Go(func() error {
			for {
				result, err := provider2.Down(context.Background())
				if err != nil {
					if errors.Is(err, goose.ErrNoCurrentVersion) {
						return nil
					}
					return err
				}
				check.NoError(t, err)
				mu.Lock()
				applied = append(applied, result.Migration.Version)
				mu.Unlock()
			}
		})
		check.NoError(t, g.Wait())
		check.Number(t, len(applied), len(migrations))
		sort.Slice(applied, func(i, j int) bool {
			return applied[i] < applied[j]
		})
		// Each migration should have been applied down exactly once. Since this is sequential the
		// applied down migrations should be in reverse order.
		for i := len(migrations) - 1; i >= 0; i-- {
			check.Number(t, applied[i], migrations[i].Version)
		}
	})
}
