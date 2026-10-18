package lock_test

import (
	"context"
	"database/sql"
	"log/slog"
	"math/rand/v2"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"github.com/pressly/goose/v3/lock/locktesting"
	"github.com/pressly/goose/v3/testdata"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// newSQLiteDB returns a file-backed SQLite database. A file is required (rather than :memory:) so
// that every connection in the pool sees the same database.
func newSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "sqlite.db") + "?_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

func TestSQLiteConcurrentTableLocking(t *testing.T) {
	t.Parallel()

	db := newSQLiteDB(t)
	// All lockers must compete for the SAME lock ID
	lockID := rand.Int64N(1<<62) + 1

	newLocker := func(t *testing.T) lock.Locker {
		locker, err := lock.NewSQLiteTableLocker(
			lock.WithTableLockID(lockID),
			lock.WithTableHeartbeatInterval(200*time.Millisecond),
			lock.WithTableLockTimeout(50*time.Millisecond, 2),
		)
		require.NoError(t, err)
		return locker
	}

	locktesting.TestConcurrentLocking(t, db, newLocker, 1*time.Second)
}

func TestSQLiteSequentialTableLocking(t *testing.T) {
	t.Parallel()

	db := newSQLiteDB(t)
	lockID := rand.Int64N(1<<62) + 1

	locker1, err := lock.NewSQLiteTableLocker(
		lock.WithTableLockID(lockID),
		lock.WithTableLeaseDuration(2*time.Second),
		lock.WithTableHeartbeatInterval(200*time.Millisecond),
	)
	require.NoError(t, err)
	locker2, err := lock.NewSQLiteTableLocker(
		lock.WithTableLockID(lockID),
		lock.WithTableLeaseDuration(2*time.Second),
		lock.WithTableHeartbeatInterval(200*time.Millisecond),
		lock.WithTableLockTimeout(50*time.Millisecond, 4),
	)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, locker1.Lock(ctx, db))

	// Second locker should fail to acquire the lock while the first one holds it.
	ctx2, cancel := context.WithTimeout(ctx, 400*time.Millisecond)
	defer cancel()
	require.Error(t, locker2.Lock(ctx2, db))

	require.NoError(t, locker1.Unlock(ctx, db))

	// Now second locker should be able to acquire the lock.
	require.NoError(t, locker2.Lock(ctx, db))
	require.NoError(t, locker2.Unlock(ctx, db))
}

func TestSQLiteProviderLocking(t *testing.T) {
	t.Parallel()

	t.Run("sqlite_table_locker_unique", func(t *testing.T) {
		t.Parallel()

		db := newSQLiteDB(t)
		// Use the same lock ID for all providers so they compete for the same table row
		sharedLockID := rand.Int64N(1<<62) + 1

		locktesting.TestProviderLocking(t, func(t *testing.T) *goose.Provider {
			t.Helper()

			// Create a UNIQUE table-based locker instance per provider, but same lock ID
			locker, err := lock.NewSQLiteTableLocker(
				lock.WithTableLockID(sharedLockID),
				lock.WithTableLockTimeout(200*time.Millisecond, 25),
			)
			require.NoError(t, err)

			p, err := goose.NewProvider(
				goose.DialectSQLite3,
				db,
				testdata.MustMigrationsFS(),
				goose.WithLocker(locker),
				goose.WithDisableGlobalRegistry(true),
			)
			require.NoError(t, err)
			return p
		})
	})
	t.Run("sqlite_table_locker_shared", func(t *testing.T) {
		t.Parallel()

		db := newSQLiteDB(t)
		// Create a SHARED table-based locker per provider
		locker, err := lock.NewSQLiteTableLocker(
			lock.WithTableLockID(rand.Int64N(1<<62)+1),
			lock.WithTableLockTimeout(200*time.Millisecond, 25),
		)
		require.NoError(t, err)

		locktesting.TestProviderLocking(t, func(t *testing.T) *goose.Provider {
			t.Helper()

			p, err := goose.NewProvider(
				goose.DialectSQLite3,
				db,
				testdata.MustMigrationsFS(),
				goose.WithLocker(locker),
				goose.WithDisableGlobalRegistry(true),
			)
			require.NoError(t, err)
			return p
		})
	})
}

func TestSQLiteTableLocker(t *testing.T) {
	t.Parallel()

	t.Run("basic_lock_unlock", func(t *testing.T) {
		t.Parallel()

		db := newSQLiteDB(t)
		locker, err := lock.NewSQLiteTableLocker(
			lock.WithTableName("test_locks"),
			lock.WithTableLockID(rand.Int64N(1<<62)+1),
			lock.WithTableLeaseDuration(5*time.Second),
			lock.WithTableHeartbeatInterval(1*time.Second),
			lock.WithTableLockTimeout(100*time.Millisecond, 2),
		)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		require.NoError(t, locker.Lock(ctx, db))
		require.NoError(t, locker.Unlock(ctx, db))
	})

	t.Run("cleanup_stale_locks", func(t *testing.T) {
		t.Parallel()

		db := newSQLiteDB(t)
		lockID := rand.Int64N(1<<62) + 1

		locker1, err := lock.NewSQLiteTableLocker(
			lock.WithTableLockID(lockID),
			lock.WithTableLeaseDuration(1*time.Second),
			lock.WithTableHeartbeatInterval(200*time.Millisecond),
		)
		require.NoError(t, err)

		ctx := context.Background()
		// Simulate the lock holder crashing: cancel its context to stop the heartbeat without
		// releasing the lock row.
		locker1Ctx, cancelLocker1 := context.WithCancel(ctx)
		require.NoError(t, locker1.Lock(locker1Ctx, db))
		cancelLocker1()

		// Let the lease expire.
		time.Sleep(1500 * time.Millisecond)

		locker2, err := lock.NewSQLiteTableLocker(
			lock.WithTableLockID(lockID),
			lock.WithTableLeaseDuration(5*time.Second),
			lock.WithTableHeartbeatInterval(1*time.Second),
			lock.WithTableLockTimeout(100*time.Millisecond, 2),
		)
		require.NoError(t, err)

		ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		require.NoError(t, locker2.Lock(ctx2, db))
		require.NoError(t, locker2.Unlock(ctx, db))
	})

	t.Run("with_logger", func(t *testing.T) {
		t.Parallel()

		db := newSQLiteDB(t)
		var logOutput strings.Builder
		logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		locker, err := lock.NewSQLiteTableLocker(
			lock.WithTableName("test_locks_with_logger"),
			lock.WithTableLockID(rand.Int64N(1<<62)+1),
			lock.WithTableLeaseDuration(2*time.Second),
			lock.WithTableHeartbeatInterval(200*time.Millisecond),
			lock.WithTableLogger(logger),
		)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		require.NoError(t, locker.Lock(ctx, db))
		// Wait for at least one heartbeat
		time.Sleep(500 * time.Millisecond)
		require.NoError(t, locker.Unlock(ctx, db))

		logs := logOutput.String()
		require.Contains(t, logs, "successfully acquired lock")
		require.Contains(t, logs, "heartbeat updated lease")
		require.Contains(t, logs, "successfully released lock")
		require.NotContains(t, logs, "heartbeat failed to update lease")
	})
}
