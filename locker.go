package goose

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/sethvargo/go-retry"
)

var (
	// ErrLockNotImplemented is returned when the database does not support locking.
	ErrLockNotImplemented = errors.New("lock not implemented")
)

type PostgresLocker struct {
	retryLock   retry.Backoff
	retryUnlock retry.Backoff
	opts        PostgresLockerOptions
}

type PostgresLockerOptions struct {
	LockID int64
}

func NewPostgresLocker(opts PostgresLockerOptions) *PostgresLocker {
	// Retry for 60 minutes, every 2 seconds, to lock the database.
	// TODO(mf): allow users to make the duration infinite for VERY long migrations.
	retryLock := retry.WithMaxDuration(
		60*time.Minute,
		retry.NewConstant(2*time.Second),
	)
	// Retry for 1 minute, every 2 seconds, to unlock the database.
	retryUnlock := retry.WithMaxDuration(
		1*time.Minute,
		retry.NewConstant(2*time.Second),
	)

	return &PostgresLocker{
		retryLock:   retryLock,
		retryUnlock: retryUnlock,
		opts:        opts,
	}
}

func (p *PostgresLocker) CanLock() bool {
	return true
}

func (p PostgresLocker) lockID() int64 {
	if p.opts.LockID != 0 {
		return p.opts.LockID
	}

	// defaultLockID is the id used to lock the database for migrations. It is a crc64 hash of the
	// string "goose". This is used to ensure that the lock is unique to goose.
	//
	// crc64.Checksum([]byte("goose"), crc64.MakeTable(crc64.ECMA))
	return 5887940537704921958
}

func (p *PostgresLocker) LockSession(ctx context.Context, conn *sql.Conn) error {
	return retry.Do(ctx, p.retryLock, func(ctx context.Context) error {
		row := conn.QueryRowContext(ctx, p.TryAdvisoryLockSession(p.lockID()))
		var locked bool
		if err := row.Scan(&locked); err != nil {
			return err
		}
		if locked {
			// A session-level advisory lock was acquired.
			return nil
		}
		// A session-level advisory lock could not be acquired. This is likely because another
		// process has already acquired the lock. We will continue retrying until the lock is
		// acquired or the maximum number of retries is reached.
		return retry.RetryableError(errors.New("failed to acquire lock"))
	})
}

func (p *PostgresLocker) UnlockSession(ctx context.Context, conn *sql.Conn) error {
	return retry.Do(ctx, p.retryUnlock, func(ctx context.Context) error {
		var unlocked bool
		row := conn.QueryRowContext(ctx, p.AdvisoryUnlockSession(p.lockID()))
		if err := row.Scan(&unlocked); err != nil {
			return err
		}
		if !unlocked {
			/*
				TODO(mf): provide the user with some documentation on how they can unlock the
				session manually.

				This is probably not an issue for 99.99% of users since pg_advisory_unlock_all()
				will release all session level advisory locks held by the current session. This
				postgres function is implicitly invoked at session end, even if the client
				disconnects ungracefully.

				Here is output from a session that has a lock held:

				SELECT pid,granted,((classid::bigint<<32)|objid::bigint)AS goose_lock_id FROM
				pg_locks WHERE locktype='advisory';


				| pid | granted | goose_lock_id       |
				|-----|---------|---------------------|
				| 191 | t       | 5887940537704921958 |

				A forceful way to unlock the session is to terminate the backend with SIGTERM:

				SELECT pg_terminate_backend(191);

				Subsequent commands on the same connection will fail with:

				Query 1 ERROR: FATAL: terminating connection due to administrator command
			*/
			return retry.RetryableError(errors.New("failed to unlock session"))
		}
		return nil
	})
}

// AdvisoryLockSession returns the query to lock the database using an exclusive session level
// advisory lock.
func (p *PostgresLocker) AdvisoryLockSession(id int64) string {
	q := `SELECT pg_advisory_lock(%d)`
	return fmt.Sprintf(q, id)
}

func (p *PostgresLocker) TryAdvisoryLockSession(id int64) string {
	q := `SELECT pg_try_advisory_lock(%d)`
	return fmt.Sprintf(q, id)
}

// AdvisoryUnlockSession returns the query to release an exclusive session level advisory lock.
func (p *PostgresLocker) AdvisoryUnlockSession(id int64) string {
	q := `SELECT pg_advisory_unlock(%d)`
	return fmt.Sprintf(q, id)
}

// AdvisoryLockTransaction returns the query to lock the database using an exclusive transaction
// level advisory lock.
//
// The lock is automatically released at the end of the current transaction and cannot be released
// explicitly.
func (p *PostgresLocker) AdvisoryLockTransaction(id int64) string {
	q := `SELECT pg_advisory_xact_lock(%d)`
	return fmt.Sprintf(q, id)
}

// func (s *store) LockTransaction(ctx context.Context, tx *sql.Tx) error {
//  switch t := s.querier.(type) {
//  case *dialectquery.Postgres:
//      return retry.Do(ctx, s.retryLock, func(ctx context.Context) error {
//          if _, err := tx.ExecContext(ctx, t.AdvisoryLockTransaction(defaultLockID)); err != nil {
//              return retry.RetryableError(err)
//          }
//          return nil
//      })
//  }
//  return ErrLockNotImplemented
// }
