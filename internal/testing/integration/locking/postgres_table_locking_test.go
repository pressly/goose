package locking_test

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/testing/testdb"
	"github.com/pressly/goose/v3/lock"
	"github.com/pressly/goose/v3/lock/locktesting"
	"github.com/pressly/goose/v3/testdata"
	"github.com/stretchr/testify/require"
)

func TestConcurrentTableLocking(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	db, cleanup, err := testdb.NewPostgres()
	require.NoError(t, err)
	t.Cleanup(cleanup)

	// All lockers must compete for the SAME lock ID
	lockID := rand.Int64()

	newLocker := func(t *testing.T) lock.Locker {
		locker, err := lock.NewPostgresTableLocker(
			lock.WithTableLockID(lockID), // Same lock ID for all lockers!!
			lock.WithTableHeartbeatInterval(200*time.Millisecond),

			// This value is important - it controls how long a locker will keep retrying to acquire
			// the lock and must be shorter than the overall lock timeout below.

			lock.WithTableLockTimeout(50*time.Millisecond, 2), // 200ms total wait time
		)
		require.NoError(t, err)
		return locker
	}

	locktesting.TestConcurrentLocking(t, db, newLocker, 1*time.Second)
}

func TestSequentialTableLocking(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	db, cleanup, err := testdb.NewPostgres()
	require.NoError(t, err)
	t.Cleanup(cleanup)

	lockID := rand.Int64()

	// Create two lockers - first has long lease, second has short retry timeout
	locker1, err := lock.NewPostgresTableLocker(
		lock.WithTableLockID(lockID),
		lock.WithTableLeaseDuration(2*time.Second), // Long lease to ensure it doesn't expire
		lock.WithTableHeartbeatInterval(200*time.Millisecond),
	)
	require.NoError(t, err)

	locker2, err := lock.NewPostgresTableLocker(
		lock.WithTableLockID(lockID),
		lock.WithTableLeaseDuration(2*time.Second),
		lock.WithTableHeartbeatInterval(200*time.Millisecond),
		lock.WithTableLockTimeout(50*time.Millisecond, 4), // Only 200ms total timeout
	)
	require.NoError(t, err)

	ctx := context.Background()

	// First locker acquires the lock
	err = locker1.Lock(ctx, db)
	require.NoError(t, err)
	t.Log("Locker 1 acquired lock")

	// Second locker should fail to acquire the lock (will timeout after 200ms of retries)
	ctx2, cancel := context.WithTimeout(ctx, 400*time.Millisecond)
	defer cancel()
	err = locker2.Lock(ctx2, db)
	require.Error(t, err)
	t.Log("Locker 2 correctly failed to acquire lock")

	// First locker releases the lock
	err = locker1.Unlock(ctx, db)
	require.NoError(t, err)
	t.Log("Locker 1 released lock")

	// Now second locker should be able to acquire the lock
	err = locker2.Lock(ctx, db)
	require.NoError(t, err)
	t.Log("Locker 2 acquired lock after locker 1 released")

	// Clean up
	err = locker2.Unlock(ctx, db)
	require.NoError(t, err)
	t.Log("Locker 2 released lock")
}

func TestLockerImplementations(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("postgres_table_locker_unique", func(t *testing.T) {
		t.Parallel()

		db, cleanup, err := testdb.NewPostgres()
		require.NoError(t, err)
		t.Cleanup(cleanup)

		// Use the same lock ID for all providers so they compete for the same table row
		sharedLockID := rand.Int64()

		locktesting.TestProviderLocking(t, func(t *testing.T) *goose.Provider {
			t.Helper()

			// Create a UNIQUE table-based locker instance per provider, but same lock ID
			locker, err := lock.NewPostgresTableLocker(
				lock.WithTableLockID(sharedLockID),
				lock.WithTableLockTimeout(200*time.Millisecond, 25), // 25 retries, 5s total
			)
			require.NoError(t, err)

			p, err := goose.NewProvider(
				goose.DialectPostgres,
				db,
				testdata.MustMigrationsFS(),
				goose.WithLocker(locker),
			)
			require.NoError(t, err)
			return p
		})
	})
	t.Run("postgres_table_locker_shared", func(t *testing.T) {
		t.Parallel()

		db, cleanup, err := testdb.NewPostgres()
		require.NoError(t, err)
		t.Cleanup(cleanup)

		// Create a SHARED table-based locker per provider
		locker, err := lock.NewPostgresTableLocker(
			lock.WithTableLockID(rand.Int64()),
			lock.WithTableLockTimeout(200*time.Millisecond, 25), // 25 retries, 5s total
		)
		require.NoError(t, err)

		locktesting.TestProviderLocking(t, func(t *testing.T) *goose.Provider {
			t.Helper()

			p, err := goose.NewProvider(
				goose.DialectPostgres,
				db,
				testdata.MustMigrationsFS(),
				goose.WithLocker(locker),
			)
			require.NoError(t, err)
			return p
		})
	})
	t.Run("postgres_session_locker_unique", func(t *testing.T) {
		t.Parallel()

		db, cleanup, err := testdb.NewPostgres()
		require.NoError(t, err)
		t.Cleanup(cleanup)

		// Use the same lock ID for all providers so they compete for the same advisory lock
		sharedLockID := rand.Int64()

		locktesting.TestProviderLocking(t, func(t *testing.T) *goose.Provider {
			t.Helper()

			// Each provider gets a UNIQUE session locker instance, but same lock ID
			// This simulates multiple pods with separate locker instances competing for same advisory lock
			sessionLocker, err := lock.NewPostgresSessionLocker(
				lock.WithLockID(sharedLockID), // Same lock ID for all providers
				lock.WithLockTimeout(1, 10),   // 10 retries, 10s total
			)
			require.NoError(t, err)

			p, err := goose.NewProvider(
				goose.DialectPostgres,
				db,
				testdata.MustMigrationsFS(),
				goose.WithSessionLocker(sessionLocker),
			)
			require.NoError(t, err)
			return p
		})
	})
	t.Run("postgres_session_locker_shared", func(t *testing.T) {
		t.Parallel()

		db, cleanup, err := testdb.NewPostgres()
		require.NoError(t, err)
		t.Cleanup(cleanup)

		// Create a shared session locker (advisory lock) for all providers
		sessionLocker, err := lock.NewPostgresSessionLocker(
			lock.WithLockID(rand.Int64()),
			lock.WithLockTimeout(1, 10), // 10 retries, 10s total
		)
		require.NoError(t, err)

		locktesting.TestProviderLocking(t, func(t *testing.T) *goose.Provider {
			t.Helper()

			p, err := goose.NewProvider(
				goose.DialectPostgres,
				db,
				testdata.MustMigrationsFS(),
				goose.WithSessionLocker(sessionLocker),
			)
			require.NoError(t, err)
			return p
		})
	})
}

func TestPostgresTableLockerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	db, cleanup, err := testdb.NewPostgres()
	require.NoError(t, err)
	t.Cleanup(cleanup)

	t.Run("basic_lock_unlock", func(t *testing.T) {
		t.Parallel()

		// Create a table locker with very short timeouts for testing
		locker, err := lock.NewPostgresTableLocker(
			lock.WithTableName("test_locks"),
			lock.WithTableLockID(rand.Int64()),
			lock.WithTableLeaseDuration(5*time.Second),
			lock.WithTableHeartbeatInterval(1*time.Second),
			lock.WithTableLockTimeout(100*time.Millisecond, 2), // Very short timeout
		)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = locker.Lock(ctx, db)
		require.NoError(t, err)

		err = locker.Unlock(ctx, db)
		require.NoError(t, err)
	})

	t.Run("cleanup_stale_locks", func(t *testing.T) {
		t.Parallel()

		lockID := rand.Int64()

		// Create a locker with very short lease to test cleanup functionality
		locker, err := lock.NewPostgresTableLocker(
			lock.WithTableLockID(lockID),
			lock.WithTableLeaseDuration(100*time.Millisecond), // Very short lease
			lock.WithTableHeartbeatInterval(50*time.Millisecond),
		)
		require.NoError(t, err)

		ctx := context.Background()

		// Acquire the lock
		err = locker.Lock(ctx, db)
		require.NoError(t, err)

		// Let the lease expire by waiting longer than lease duration
		time.Sleep(200 * time.Millisecond)

		// Create a second locker that should be able to acquire the lock
		// because the first one's lease has expired
		locker2, err := lock.NewPostgresTableLocker(
			lock.WithTableLockID(lockID), // Same lock ID
			lock.WithTableLeaseDuration(5*time.Second),
			lock.WithTableHeartbeatInterval(1*time.Second),
		)
		require.NoError(t, err)

		// This should succeed because cleanup of stale locks allows it
		ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		err = locker2.Lock(ctx2, db)
		require.NoError(t, err)

		// Clean up
		err = locker2.Unlock(ctx, db)
		require.NoError(t, err)
	})

	t.Run("with_logger", func(t *testing.T) {
		t.Parallel()

		var logOutput strings.Builder
		logger := slog.New(slog.NewTextHandler(&logOutput, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		// Create a table locker with logging enabled
		locker, err := lock.NewPostgresTableLocker(
			lock.WithTableName("test_locks_with_logger"),
			lock.WithTableLockID(rand.Int64()),
			lock.WithTableLeaseDuration(2*time.Second),
			lock.WithTableHeartbeatInterval(500*time.Millisecond),
			lock.WithTableLogger(logger),
		)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Test that lock operations generate log messages
		err = locker.Lock(ctx, db)
		require.NoError(t, err)

		// Wait a moment for heartbeat
		time.Sleep(1 * time.Second)

		err = locker.Unlock(ctx, db)
		require.NoError(t, err)

		// Check that we got some log output
		logs := logOutput.String()
		require.Contains(t, logs, "successfully acquired lock")
		require.Contains(t, logs, "successfully released lock")
		require.Contains(t, logs, "heartbeat updated lease")
	})
}
