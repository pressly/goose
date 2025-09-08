package lock

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestNewTableSessionLocker(t *testing.T) {
	t.Run("default_config", func(t *testing.T) {
		locker, err := newSQLiteTableSessionLocker()
		require.NoError(t, err)
		require.NotNil(t, locker)

		tl := locker.(*tableSessionLocker)
		require.Equal(t, DefaultLockID, tl.lockID)
		require.Equal(t, 30*time.Second, tl.heartbeatInterval)
		require.Equal(t, 5*time.Minute, tl.staleTimeout)
		require.Contains(t, tl.processInfo, ":")
	})

	t.Run("custom_config", func(t *testing.T) {
		locker, err := newSQLiteTableSessionLocker(
			WithLockID(12345),
			WithHeartbeatInterval(10*time.Second),
			WithStaleTimeout(2*time.Minute),
			WithLockTimeout(2, 5),
			WithUnlockTimeout(1, 3),
		)
		require.NoError(t, err)

		tl := locker.(*tableSessionLocker)
		require.Equal(t, int64(12345), tl.lockID)
		require.Equal(t, 10*time.Second, tl.heartbeatInterval)
		require.Equal(t, 2*time.Minute, tl.staleTimeout)
	})

	t.Run("invalid_heartbeat_interval", func(t *testing.T) {
		_, err := newSQLiteTableSessionLocker(
			WithHeartbeatInterval(500 * time.Millisecond),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "heartbeat interval must be at least 1 second")
	})

	t.Run("invalid_stale_timeout", func(t *testing.T) {
		_, err := newSQLiteTableSessionLocker(
			WithStaleTimeout(30 * time.Second),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "stale timeout must be at least 1 minute")
	})
}

func TestTableSessionLocker_BasicLocking(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()

	locker, err := newSQLiteTableSessionLocker(
		WithLockTimeout(1, 3), // Quick timeout for testing
		WithUnlockTimeout(1, 3),
		WithHeartbeatInterval(time.Second),
	)
	require.NoError(t, err)

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	t.Run("lock_and_unlock", func(t *testing.T) {
		// Acquire lock
		err := locker.SessionLock(ctx, conn)
		require.NoError(t, err)

		// Verify lock exists in database
		exists, err := lockExists(ctx, conn)
		require.NoError(t, err)
		require.True(t, exists, "lock should exist after acquiring")

		// Release lock
		err = locker.SessionUnlock(ctx, conn)
		require.NoError(t, err)

		// Verify lock is released
		exists, err = lockExists(ctx, conn)
		require.NoError(t, err)
		require.False(t, exists, "lock should be released")
	})

	t.Run("double_lock_same_connection", func(t *testing.T) {
		// First lock should succeed
		err := locker.SessionLock(ctx, conn)
		require.NoError(t, err)
		defer locker.SessionUnlock(ctx, conn)

		// Second lock on same connection should fail quickly
		err = locker.SessionLock(ctx, conn)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to acquire lock")
	})
}

func TestTableSessionLocker_ConcurrentLocking(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()

	ctx := context.Background()
	const numWorkers = 5

	t.Run("exactly_one_winner", func(t *testing.T) {
		var wg sync.WaitGroup
		var mu sync.Mutex
		var successes, failures int

		// Use a single shared database
		sharedDB, cleanup := newTestDB(t)
		defer cleanup()

		for i := range numWorkers {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				locker, err := newSQLiteTableSessionLocker(
					WithLockTimeout(1, 2), // Quick timeout
					WithHeartbeatInterval(time.Second),
				)
				require.NoError(t, err)

				conn, err := sharedDB.Conn(ctx)
				require.NoError(t, err)
				defer conn.Close()

				err = locker.SessionLock(ctx, conn)
				
				mu.Lock()
				if err != nil {
					failures++
				} else {
					successes++
					// Hold lock briefly then release
					time.Sleep(100 * time.Millisecond)
					locker.SessionUnlock(ctx, conn)
				}
				mu.Unlock()
			}(i)
		}

		wg.Wait()

		// Exactly one should succeed, others should fail
		require.Equal(t, 1, successes, "exactly one worker should acquire the lock")
		require.Equal(t, numWorkers-1, failures, "all other workers should fail")
	})

	t.Run("sequential_locking", func(t *testing.T) {
		locker1, err := newSQLiteTableSessionLocker(WithHeartbeatInterval(time.Second))
		require.NoError(t, err)

		locker2, err := newSQLiteTableSessionLocker(WithHeartbeatInterval(time.Second))
		require.NoError(t, err)

		conn1, err := db.Conn(ctx)
		require.NoError(t, err)
		defer conn1.Close()

		conn2, err := db.Conn(ctx)
		require.NoError(t, err)
		defer conn2.Close()

		// First locker acquires lock
		err = locker1.SessionLock(ctx, conn1)
		require.NoError(t, err)

		// Second locker should fail to acquire
		err = locker2.SessionLock(ctx, conn2)
		require.Error(t, err)

		// First locker releases
		err = locker1.SessionUnlock(ctx, conn1)
		require.NoError(t, err)

		// Now second locker should succeed
		err = locker2.SessionLock(ctx, conn2)
		require.NoError(t, err)

		// Clean up
		err = locker2.SessionUnlock(ctx, conn2)
		require.NoError(t, err)
	})
}

func TestTableSessionLocker_HeartbeatAndStaleDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping heartbeat test in short mode")
	}

	db, cleanup := newTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("heartbeat_updates", func(t *testing.T) {
		locker, err := newSQLiteTableSessionLocker(
			WithHeartbeatInterval(time.Second), // Fast heartbeat for testing  
			WithStaleTimeout(2 * time.Minute),
		)
		require.NoError(t, err)

		conn, err := db.Conn(ctx)
		require.NoError(t, err)
		defer conn.Close()

		// Acquire lock
		err = locker.SessionLock(ctx, conn)
		require.NoError(t, err)
		defer locker.SessionUnlock(ctx, conn)

		// Get initial heartbeat
		heartbeat1, err := getHeartbeat(ctx, conn)
		require.NoError(t, err)

		// Wait for heartbeat to update (wait longer than heartbeat interval)
		time.Sleep(2 * time.Second)

		// Get updated heartbeat
		heartbeat2, err := getHeartbeat(ctx, conn)
		require.NoError(t, err)

		// Heartbeat should have been updated
		require.True(t, heartbeat2.After(heartbeat1), "heartbeat should be updated")
	})

	t.Run("stale_lock_cleanup", func(t *testing.T) {
		// Create a locker with very short stale timeout for testing
		locker1, err := newSQLiteTableSessionLocker(
			WithStaleTimeout(time.Minute),
			WithHeartbeatInterval(time.Second),
		)
		require.NoError(t, err)

		conn1, err := db.Conn(ctx)
		require.NoError(t, err)
		defer conn1.Close()

		// Acquire lock
		err = locker1.SessionLock(ctx, conn1)
		require.NoError(t, err)

		// Simulate stale lock by manually updating the heartbeat to be very old
		err = makeStale(ctx, conn1)
		require.NoError(t, err)

		// Create second locker that should clean up stale lock
		locker2, err := newSQLiteTableSessionLocker(
			WithStaleTimeout(time.Minute),
			WithLockTimeout(1, 3), // Quick timeout
		)
		require.NoError(t, err)

		conn2, err := db.Conn(ctx)
		require.NoError(t, err)
		defer conn2.Close()

		// Second locker should clean up stale lock and acquire it
		err = locker2.SessionLock(ctx, conn2)
		require.NoError(t, err)

		// Verify second locker has the lock
		exists, err := lockExists(ctx, conn2)
		require.NoError(t, err)
		require.True(t, exists)

		// Clean up
		locker2.SessionUnlock(ctx, conn2)
	})
}

func TestTableSessionLocker_ErrorHandling(t *testing.T) {
	db, cleanup := newTestDB(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("unlock_without_lock", func(t *testing.T) {
		locker, err := newSQLiteTableSessionLocker(WithUnlockTimeout(1, 2))
		require.NoError(t, err)

		conn, err := db.Conn(ctx)
		require.NoError(t, err)
		defer conn.Close()

		// Try to unlock without holding lock
		err = locker.SessionUnlock(ctx, conn)
		require.Error(t, err)
		// Error can be either "failed to release lock" or table doesn't exist
		require.True(t, err.Error() == "failed to release lock" || 
			strings.Contains(err.Error(), "no such table"))
	})

	t.Run("context_cancellation", func(t *testing.T) {
		locker, err := newSQLiteTableSessionLocker(
			WithLockTimeout(1, 10), // Long timeout
		)
		require.NoError(t, err)

		conn, err := db.Conn(ctx)
		require.NoError(t, err)
		defer conn.Close()

		// First acquire lock in another connection
		locker2, err := newSQLiteTableSessionLocker()
		require.NoError(t, err)
		conn2, err := db.Conn(ctx)
		require.NoError(t, err)
		defer conn2.Close()

		err = locker2.SessionLock(ctx, conn2)
		require.NoError(t, err)
		defer locker2.SessionUnlock(ctx, conn2)

		// Cancel context while waiting for lock
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel()

		err = locker.SessionLock(cancelCtx, conn)
		require.Error(t, err)
		require.True(t, errors.Is(err, context.Canceled) || 
			errors.Is(err, context.DeadlineExceeded))
	})
}

// Helper functions

func newSQLiteTableSessionLocker(opts ...SessionLockerOption) (SessionLocker, error) {
	// Create SQLite-specific lock store for testing
	store := NewLockStore(DefaultLockTableName, NewSQLiteLockQuerier())
	return NewTableSessionLockerWithStore(store, opts...)
}

func newTestDB(t *testing.T) (*sql.DB, func()) {
	// Use a temporary file database for proper concurrency testing
	tmpfile := t.TempDir() + "/test.db"
	db, err := sql.Open("sqlite", tmpfile)
	require.NoError(t, err)

	// Enable WAL mode and immediate locking for better concurrency
	_, err = db.Exec("PRAGMA journal_mode=WAL; PRAGMA locking_mode=NORMAL; PRAGMA synchronous=NORMAL;")
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func lockExists(ctx context.Context, conn *sql.Conn) (bool, error) {
	var locked int
	err := conn.QueryRowContext(ctx, "SELECT locked FROM "+DefaultLockTableName+" WHERE id = 1").Scan(&locked)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		// If table doesn't exist, no lock exists
		if strings.Contains(err.Error(), "no such table") {
			return false, nil
		}
		return false, err
	}
	return locked == 1, nil
}

func getHeartbeat(ctx context.Context, conn *sql.Conn) (time.Time, error) {
	var heartbeat string
	err := conn.QueryRowContext(ctx, "SELECT last_heartbeat FROM "+DefaultLockTableName+" WHERE id = 1").Scan(&heartbeat)
	if err != nil {
		return time.Time{}, err
	}
	
	// Parse SQLite datetime format
	return time.Parse("2006-01-02 15:04:05", heartbeat)
}

func makeStale(ctx context.Context, conn *sql.Conn) error {
	// Set heartbeat to 10 minutes ago to make it stale
	_, err := conn.ExecContext(ctx, 
		"UPDATE "+DefaultLockTableName+" SET last_heartbeat = datetime('now', '-10 minutes') WHERE id = 1")
	return err
}