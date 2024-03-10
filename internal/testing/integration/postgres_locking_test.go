package integration

import (
	"context"
	"database/sql"
	"errors"
	"math/rand"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/testing/testdb"
	"github.com/pressly/goose/v3/lock"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func TestPostgresSessionLocker(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	db, cleanup, err := testdb.NewPostgres()
	require.NoError(t, err)
	t.Cleanup(cleanup)

	// Do not run subtests in parallel, because they are using the same database.

	t.Run("lock_and_unlock", func(t *testing.T) {
		const (
			lockID int64 = 123456789
		)
		locker, err := lock.NewPostgresSessionLocker(
			lock.WithLockID(lockID),
			lock.WithLockTimeout(1, 4),   // 4 second timeout
			lock.WithUnlockTimeout(1, 4), // 4 second timeout
		)
		require.NoError(t, err)
		ctx := context.Background()
		conn, err := db.Conn(ctx)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, conn.Close())
		})
		err = locker.SessionLock(ctx, conn)
		require.NoError(t, err)
		// Check that the lock was acquired.
		exists, err := existsPgLock(ctx, db, lockID)
		require.NoError(t, err)
		require.True(t, exists)
		// Check that the lock is released.
		err = locker.SessionUnlock(ctx, conn)
		require.NoError(t, err)
		exists, err = existsPgLock(ctx, db, lockID)
		require.NoError(t, err)
		require.False(t, exists)
	})
	t.Run("lock_close_conn_unlock", func(t *testing.T) {
		locker, err := lock.NewPostgresSessionLocker(
			lock.WithLockTimeout(1, 4),   // 4 second timeout
			lock.WithUnlockTimeout(1, 4), // 4 second timeout
		)
		require.NoError(t, err)
		ctx := context.Background()
		conn, err := db.Conn(ctx)
		require.NoError(t, err)

		err = locker.SessionLock(ctx, conn)
		require.NoError(t, err)
		exists, err := existsPgLock(ctx, db, lock.DefaultLockID)
		require.NoError(t, err)
		require.True(t, exists)
		// Simulate a connection close.
		err = conn.Close()
		require.NoError(t, err)
		// Check an error is returned when unlocking, because the connection is already closed.
		err = locker.SessionUnlock(ctx, conn)
		require.Error(t, err)
		require.True(t, errors.Is(err, sql.ErrConnDone))
	})
	t.Run("multiple_connections", func(t *testing.T) {
		const (
			workers = 5
		)
		ch := make(chan error)
		var wg sync.WaitGroup
		for i := 0; i < workers; i++ {
			wg.Add(1)

			go func() {
				defer wg.Done()
				ctx := context.Background()
				conn, err := db.Conn(ctx)
				require.NoError(t, err)
				t.Cleanup(func() {
					require.NoError(t, conn.Close())
				})
				// Exactly one connection should acquire the lock. While the other connections
				// should fail to acquire the lock and timeout.
				locker, err := lock.NewPostgresSessionLocker(
					lock.WithLockTimeout(1, 4),   // 4 second timeout
					lock.WithUnlockTimeout(1, 4), // 4 second timeout
				)
				require.NoError(t, err)
				// NOTE, we are not unlocking the lock, because we want to test that the lock is
				// released when the connection is closed.
				ch <- locker.SessionLock(ctx, conn)
			}()
		}
		go func() {
			wg.Wait()
			close(ch)
		}()
		var errors []error
		for err := range ch {
			if err != nil {
				errors = append(errors, err)
			}
		}
		require.Equal(t, len(errors), workers-1) // One worker succeeds, the rest fail.
		for _, err := range errors {
			require.Error(t, err)
			require.Equal(t, err.Error(), "failed to acquire lock")
		}
		exists, err := existsPgLock(context.Background(), db, lock.DefaultLockID)
		require.NoError(t, err)
		require.True(t, exists)
	})
	t.Run("unlock_with_different_connection_error", func(t *testing.T) {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		randomLockID := rng.Int63n(90000) + 10000
		ctx := context.Background()
		locker, err := lock.NewPostgresSessionLocker(
			lock.WithLockID(randomLockID),
			lock.WithLockTimeout(1, 4),   // 4 second timeout
			lock.WithUnlockTimeout(1, 4), // 4 second timeout
		)
		require.NoError(t, err)

		conn1, err := db.Conn(ctx)
		require.NoError(t, err)
		err = locker.SessionLock(ctx, conn1)
		require.NoError(t, err)
		t.Cleanup(func() {
			// Defer the unlock with the same connection.
			err = locker.SessionUnlock(ctx, conn1)
			require.NoError(t, err)
			require.NoError(t, conn1.Close())
		})
		exists, err := existsPgLock(ctx, db, randomLockID)
		require.NoError(t, err)
		require.True(t, exists)
		// Unlock with a different connection.
		conn2, err := db.Conn(ctx)
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, conn2.Close())
		})
		// Check an error is returned when unlocking with a different connection.
		err = locker.SessionUnlock(ctx, conn2)
		require.Error(t, err)
	})
}

func TestPostgresProviderLocking(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	// The migrations are written in such a way they cannot be applied in parallel, they will fail
	// 99.9999% of the time. This test ensures that the advisory session lock mode works as
	// expected.

	// TODO(mf): small improvement here is to use the SAME postgres instance but different databases
	// created from a template. This will speed up the test.

	db, cleanup, err := testdb.NewPostgres()
	require.NoError(t, err)
	t.Cleanup(cleanup)

	newProvider := func() *goose.Provider {

		sessionLocker, err := lock.NewPostgresSessionLocker(
			lock.WithLockTimeout(5, 60), // Timeout 5min. Try every 5s up to 60 times.
		)
		require.NoError(t, err)
		p, err := goose.NewProvider(
			goose.DialectPostgres,
			db,
			os.DirFS("testdata/migrations/postgres"),
			goose.WithSessionLocker(sessionLocker), // Use advisory session lock mode.
		)
		require.NoError(t, err)

		return p
	}

	provider1 := newProvider()
	provider2 := newProvider()

	sources := provider1.ListSources()
	maxVersion := sources[len(sources)-1].Version

	// Since the lock mode is advisory session, only one of these providers is expected to apply ALL
	// the migrations. The other provider should apply NO migrations. The test MUST fail if both
	// providers apply migrations.

	t.Run("up", func(t *testing.T) {
		var g errgroup.Group
		var res1, res2 int
		g.Go(func() error {
			ctx := context.Background()
			results, err := provider1.Up(ctx)
			require.NoError(t, err)
			res1 = len(results)
			currentVersion, err := provider1.GetDBVersion(ctx)
			require.NoError(t, err)
			require.Equal(t, currentVersion, maxVersion)
			return nil
		})
		g.Go(func() error {
			ctx := context.Background()
			results, err := provider2.Up(ctx)
			require.NoError(t, err)
			res2 = len(results)
			currentVersion, err := provider2.GetDBVersion(ctx)
			require.NoError(t, err)
			require.Equal(t, currentVersion, maxVersion)
			return nil
		})
		require.NoError(t, g.Wait())
		// One of the providers should have applied all migrations and the other should have applied
		// no migrations, but with no error.
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
		_, err := provider1.DownTo(context.Background(), 0)
		require.NoError(t, err)
		currentVersion, err := provider1.GetDBVersion(context.Background())
		require.NoError(t, err)
		require.Equal(t, currentVersion, int64(0))
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
				require.NoError(t, err)
				require.NotNil(t, result)
				mu.Lock()
				applied = append(applied, result.Source.Version)
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
				require.NoError(t, err)
				require.NotNil(t, result)
				mu.Lock()
				applied = append(applied, result.Source.Version)
				mu.Unlock()
			}
		})
		require.NoError(t, g.Wait())
		require.Equal(t, len(applied), len(sources))
		sort.Slice(applied, func(i, j int) bool {
			return applied[i] < applied[j]
		})
		// Each migration should have been applied up exactly once.
		for i := 0; i < len(sources); i++ {
			require.Equal(t, applied[i], sources[i].Version)
		}
	})

	// Restore the database state by applying all migrations and run the same test with the advisory
	// lock mode, but apply down migrations in parallel.
	{
		_, err := provider1.Up(context.Background())
		require.NoError(t, err)
		currentVersion, err := provider1.GetDBVersion(context.Background())
		require.NoError(t, err)
		require.Equal(t, currentVersion, maxVersion)
	}

	t.Run("down_to", func(t *testing.T) {
		var g errgroup.Group
		var res1, res2 int
		g.Go(func() error {
			ctx := context.Background()
			results, err := provider1.DownTo(ctx, 0)
			require.NoError(t, err)
			res1 = len(results)
			currentVersion, err := provider1.GetDBVersion(ctx)
			require.NoError(t, err)
			require.Equal(t, int64(0), currentVersion)
			return nil
		})
		g.Go(func() error {
			ctx := context.Background()
			results, err := provider2.DownTo(ctx, 0)
			require.NoError(t, err)
			res2 = len(results)
			currentVersion, err := provider2.GetDBVersion(ctx)
			require.NoError(t, err)
			require.Equal(t, int64(0), currentVersion)
			return nil
		})
		require.NoError(t, g.Wait())

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
		require.NoError(t, err)
		currentVersion, err := provider1.GetDBVersion(context.Background())
		require.NoError(t, err)
		require.Equal(t, currentVersion, maxVersion)
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
					if errors.Is(err, goose.ErrNoNextVersion) {
						return nil
					}
					return err
				}
				require.NoError(t, err)
				require.NotNil(t, result)
				mu.Lock()
				applied = append(applied, result.Source.Version)
				mu.Unlock()
			}
		})
		g.Go(func() error {
			for {
				result, err := provider2.Down(context.Background())
				if err != nil {
					if errors.Is(err, goose.ErrNoNextVersion) {
						return nil
					}
					return err
				}
				require.NoError(t, err)
				require.NotNil(t, result)
				mu.Lock()
				applied = append(applied, result.Source.Version)
				mu.Unlock()
			}
		})
		require.NoError(t, g.Wait())
		require.Equal(t, len(applied), len(sources))
		sort.Slice(applied, func(i, j int) bool {
			return applied[i] < applied[j]
		})
		// Each migration should have been applied down exactly once. Since this is sequential the
		// applied down migrations should be in reverse order.
		for i := len(sources) - 1; i >= 0; i-- {
			require.Equal(t, applied[i], sources[i].Version)
		}
	})
}

func existsPgLock(ctx context.Context, db *sql.DB, lockID int64) (bool, error) {
	q := `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE locktype='advisory' AND ((classid::bigint<<32)|objid::bigint)=$1)`
	row := db.QueryRowContext(ctx, q, lockID)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}
