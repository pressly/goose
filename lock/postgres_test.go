package lock_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/pressly/goose/v3/internal/check"
	"github.com/pressly/goose/v3/internal/testdb"
	"github.com/pressly/goose/v3/lock"
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
			lock.WithLockTimeout(4*time.Second),
			lock.WithUnlockTimeout(4*time.Second),
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
		pgLocks, err := queryPgLocks(ctx, db)
		check.NoError(t, err)
		check.Number(t, len(pgLocks), 1)
		// Check that the lock was acquired.
		check.Bool(t, pgLocks[0].granted, true)
		// Check that the custom lock ID is the same as the one used by the locker.
		check.Equal(t, pgLocks[0].gooseLockID, lockID)
		check.NumberNotZero(t, pgLocks[0].pid)

		// Check that the lock is released.
		err = locker.SessionUnlock(ctx, conn)
		check.NoError(t, err)
		pgLocks, err = queryPgLocks(ctx, db)
		check.NoError(t, err)
		check.Number(t, len(pgLocks), 0)
	})
	t.Run("lock_close_conn_unlock", func(t *testing.T) {
		locker, err := lock.NewPostgresSessionLocker(
			lock.WithLockTimeout(4*time.Second),
			lock.WithUnlockTimeout(4*time.Second),
		)
		check.NoError(t, err)
		ctx := context.Background()
		conn, err := db.Conn(ctx)
		check.NoError(t, err)

		err = locker.SessionLock(ctx, conn)
		check.NoError(t, err)
		pgLocks, err := queryPgLocks(ctx, db)
		check.NoError(t, err)
		check.Number(t, len(pgLocks), 1)
		check.Bool(t, pgLocks[0].granted, true)
		check.Equal(t, pgLocks[0].gooseLockID, lock.DefaultLockID)
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
					lock.WithLockTimeout(4*time.Second),
					lock.WithUnlockTimeout(4*time.Second),
				)
				check.NoError(t, err)
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
		pgLocks, err := queryPgLocks(context.Background(), db)
		check.NoError(t, err)
		check.Number(t, len(pgLocks), 1)
		check.Bool(t, pgLocks[0].granted, true)
		check.Equal(t, pgLocks[0].gooseLockID, lock.DefaultLockID)
	})
	t.Run("unlock_with_different_connection", func(t *testing.T) {
		ctx := context.Background()
		const (
			lockID int64 = 999
		)
		locker, err := lock.NewPostgresSessionLocker(
			lock.WithLockID(lockID),
			lock.WithLockTimeout(4*time.Second),
			lock.WithUnlockTimeout(4*time.Second),
		)
		check.NoError(t, err)

		conn1, err := db.Conn(ctx)
		check.NoError(t, err)
		t.Cleanup(func() {
			check.NoError(t, conn1.Close())
		})
		err = locker.SessionLock(ctx, conn1)
		check.NoError(t, err)
		pgLocks, err := queryPgLocks(ctx, db)
		check.NoError(t, err)
		check.Number(t, len(pgLocks), 1)
		check.Bool(t, pgLocks[0].granted, true)
		check.Equal(t, pgLocks[0].gooseLockID, lockID)
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

type pgLock struct {
	pid         int
	granted     bool
	gooseLockID int64
}

func queryPgLocks(ctx context.Context, db *sql.DB) ([]pgLock, error) {
	q := `SELECT pid,granted,((classid::bigint<<32)|objid::bigint)AS goose_lock_id FROM pg_locks WHERE locktype='advisory'`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	var pgLocks []pgLock
	for rows.Next() {
		var p pgLock
		if err = rows.Scan(&p.pid, &p.granted, &p.gooseLockID); err != nil {
			return nil, err
		}
		pgLocks = append(pgLocks, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return pgLocks, nil
}
