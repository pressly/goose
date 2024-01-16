package lock_test

import (
	"context"
	"database/sql"
	"errors"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/piiano/goose/v3/internal/check"
	"github.com/piiano/goose/v3/internal/testdb"
	"github.com/piiano/goose/v3/lock"
)

func TestPostgresSessionLocker(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skip long running test")
	}
	db, cleanup, err := testdb.NewPostgres()
	check.NoError(t, err)
	t.Cleanup(cleanup)

	// Do not run tests in parallel, because they are using the same database.

	t.Run("lock_and_unlock", func(t *testing.T) {
		const (
			lockID int64 = 123456789
		)
		locker, err := lock.NewPostgresSessionLocker(
			lock.WithLockID(lockID),
			lock.WithLockTimeout(1, 4),   // 4 second timeout
			lock.WithUnlockTimeout(1, 4), // 4 second timeout
		)
		check.NoError(t, err)
		ctx := context.Background()
		conn, err := db.Conn(ctx)
		check.NoError(t, err)
		t.Cleanup(func() {
			check.NoError(t, conn.Close())
		})
		err = locker.SessionLock(ctx, conn)
		check.NoError(t, err)
		// Check that the lock was acquired.
		exists, err := existsPgLock(ctx, db, lockID)
		check.NoError(t, err)
		check.Bool(t, exists, true)
		// Check that the lock is released.
		err = locker.SessionUnlock(ctx, conn)
		check.NoError(t, err)
		exists, err = existsPgLock(ctx, db, lockID)
		check.NoError(t, err)
		check.Bool(t, exists, false)
	})
	t.Run("lock_close_conn_unlock", func(t *testing.T) {
		locker, err := lock.NewPostgresSessionLocker(
			lock.WithLockTimeout(1, 4),   // 4 second timeout
			lock.WithUnlockTimeout(1, 4), // 4 second timeout
		)
		check.NoError(t, err)
		ctx := context.Background()
		conn, err := db.Conn(ctx)
		check.NoError(t, err)

		err = locker.SessionLock(ctx, conn)
		check.NoError(t, err)
		exists, err := existsPgLock(ctx, db, lock.DefaultLockID)
		check.NoError(t, err)
		check.Bool(t, exists, true)
		// Simulate a connection close.
		err = conn.Close()
		check.NoError(t, err)
		// Check an error is returned when unlocking, because the connection is already closed.
		err = locker.SessionUnlock(ctx, conn)
		check.HasError(t, err)
		check.Bool(t, errors.Is(err, sql.ErrConnDone), true)
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
				check.NoError(t, err)
				t.Cleanup(func() {
					check.NoError(t, conn.Close())
				})
				// Exactly one connection should acquire the lock. While the other connections
				// should fail to acquire the lock and timeout.
				locker, err := lock.NewPostgresSessionLocker(
					lock.WithLockTimeout(1, 4),   // 4 second timeout
					lock.WithUnlockTimeout(1, 4), // 4 second timeout
				)
				check.NoError(t, err)
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
		check.Equal(t, len(errors), workers-1) // One worker succeeds, the rest fail.
		for _, err := range errors {
			check.HasError(t, err)
			check.Equal(t, err.Error(), "failed to acquire lock")
		}
		exists, err := existsPgLock(context.Background(), db, lock.DefaultLockID)
		check.NoError(t, err)
		check.Bool(t, exists, true)
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
		check.NoError(t, err)

		conn1, err := db.Conn(ctx)
		check.NoError(t, err)
		err = locker.SessionLock(ctx, conn1)
		check.NoError(t, err)
		t.Cleanup(func() {
			// Defer the unlock with the same connection.
			err = locker.SessionUnlock(ctx, conn1)
			check.NoError(t, err)
			check.NoError(t, conn1.Close())
		})
		exists, err := existsPgLock(ctx, db, randomLockID)
		check.NoError(t, err)
		check.Bool(t, exists, true)
		// Unlock with a different connection.
		conn2, err := db.Conn(ctx)
		check.NoError(t, err)
		t.Cleanup(func() {
			check.NoError(t, conn2.Close())
		})
		// Check an error is returned when unlocking with a different connection.
		err = locker.SessionUnlock(ctx, conn2)
		check.HasError(t, err)
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
