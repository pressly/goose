package dialectadapter

import (
	"context"
	"database/sql"
	"errors"

	"github.com/pressly/goose/v4/internal/dialectadapter/dialectquery"
	"github.com/sethvargo/go-retry"
)

var (
	// defaultLockID is the id used to lock the database for migrations. It is a crc64 hash of the
	// string "goose". This is used to ensure that the lock is unique to goose.
	//
	// crc64.Checksum([]byte("goose"), crc64.MakeTable(crc64.ECMA))
	defaultLockID int64 = 5887940537704921958

	// ErrLockNotImplemented is returned when the database does not support locking.
	ErrLockNotImplemented = errors.New("lock not implemented")
)

// Locker defines the methods to lock and unlock the database.
//
// Locking is an experimental feature and the underlying implementation may change in the future.
//
// The only database that currently supports locking is Postgres. Other databases will return
// ErrLockNotImplemented.
type Locker interface {
	// CanLock returns true if the database supports locking.
	CanLock() bool

	// LockSession and UnlockSession are used to lock the database for the duration of a session.
	//
	// The session is defined as the duration of a single connection and both methods must be called
	// on the same connection.
	LockSession(ctx context.Context, conn *sql.Conn) error
	UnlockSession(ctx context.Context, conn *sql.Conn) error

	// LockTransaction is used to lock the database for the duration of a transaction.
	// LockTransaction(ctx context.Context, tx *sql.Tx) error
}

func (s *store) CanLock() bool {
	switch s.querier.(type) {
	case *dialectquery.Postgres:
		return true
	default:
		return false
	}
}

func (s *store) LockSession(ctx context.Context, conn *sql.Conn) error {
	var fn func(context.Context) error

	switch t := s.querier.(type) {
	case *dialectquery.Postgres:
		fn = func(ctx context.Context) error {
			row := conn.QueryRowContext(ctx, t.TryAdvisoryLockSession(defaultLockID))
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
		}
	default:
		return ErrLockNotImplemented
	}
	return retry.Do(ctx, s.retryLock, fn)
}

func (s *store) UnlockSession(ctx context.Context, conn *sql.Conn) error {
	var fn func(context.Context) error

	switch t := s.querier.(type) {
	case *dialectquery.Postgres:
		fn = func(ctx context.Context) error {
			var unlocked bool
			row := conn.QueryRowContext(ctx, t.AdvisoryUnlockSession(defaultLockID))
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
		}
	default:
		return ErrLockNotImplemented
	}
	return retry.Do(ctx, s.retryUnlock, fn)
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
