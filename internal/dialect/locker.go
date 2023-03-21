package dialect

import (
	"context"
	"database/sql"
	"errors"
	"hash/crc64"

	"github.com/pressly/goose/v3/internal/dialect/dialectquery"
	"github.com/sethvargo/go-retry"
)

var (
	// defaultLockID is the id used to lock the database for migrations. It is a
	// crc64 hash of the string "goose". This is used to prevent multiple
	// goose processes from running migrations at the same time.
	//
	// 5887940537704921958
	defaultLockID = crc64.Checksum([]byte("goose"), crc64.MakeTable(crc64.ECMA))

	// ErrLockNotImplemented is returned when the database does not support locking.
	ErrLockNotImplemented = errors.New("lock not implemented")
)

// Locker defines the methods to lock and unlock the database.
//
// Locking is an experimental feature and the underlying implementation
// may change in the future.
//
// The only database that currently supports locking is Postgres.
//
// Other databases will return ErrLockNotImplemented.
type Locker interface {
	// LockSession and UnlockSession are used to lock the database for the
	// duration of a session.
	//
	// The session is defined as the duration of a single connection.
	LockSession(ctx context.Context, conn *sql.Conn) error
	UnlockSession(ctx context.Context, conn *sql.Conn) error

	// LockTransaction is used to lock the database for the duration of a
	// transaction.
	LockTransaction(ctx context.Context, tx *sql.Tx) error
}

func (s *store) LockSession(ctx context.Context, conn *sql.Conn) error {
	switch t := s.querier.(type) {
	case *dialectquery.Postgres:
		return retry.Do(ctx, s.retry, func(ctx context.Context) error {
			if _, err := conn.ExecContext(ctx, t.AdvisoryLockSession(), defaultLockID); err != nil {
				return retry.RetryableError(err)
			}
			return nil
		})
	}
	return ErrLockNotImplemented
}

func (s *store) UnlockSession(ctx context.Context, conn *sql.Conn) error {
	switch t := s.querier.(type) {
	case *dialectquery.Postgres:
		return retry.Do(ctx, s.retry, func(ctx context.Context) error {
			var unlocked bool
			row := conn.QueryRowContext(ctx, t.AdvisoryUnlockSession(), defaultLockID)
			if err := row.Scan(&unlocked); err != nil {
				return retry.RetryableError(err)
			}
			if !unlocked {
				// TODO(mf): provide the user with some documentation on how they
				// can unlock the session manually. Although this might not be an issue
				// for 99.9% of users since pg_advisory_unlock_all() will release all session
				// level advisory locks held by the current session.
				// (This function is implicitly invoked at session end, even if the client disconnects ungracefully.)
				//
				// TODO(mf): - we may not want to return an error here or bother checking the return value!
				//
				// SELECT pid,granted,((classid::bigint<<32)|objid::bigint)AS goose_lock_id FROM pg_locks WHERE locktype='advisory';
				//
				// | pid | granted | goose_lock_id       |
				// |-----|---------|---------------------|
				// | 191 | t       | 5887940537704921958 |
				//
				// A graceful way to unlock the session is to kill the process:
				// SELECT pg_cancel_backend(120);
				//
				// A more forceful way to unlock the session is to terminate the process:
				// SELECT pg_terminate_backend(120);

				return errors.New("failed to unlock session")
			}
			return nil
		})
	}
	return ErrLockNotImplemented
}

func (s *store) LockTransaction(ctx context.Context, tx *sql.Tx) error {
	switch t := s.querier.(type) {
	case *dialectquery.Postgres:
		return retry.Do(ctx, s.retry, func(ctx context.Context) error {
			if _, err := tx.ExecContext(ctx, t.AdvisoryLockTransaction(), defaultLockID); err != nil {
				return retry.RetryableError(err)
			}
			return nil
		})
	}
	return ErrLockNotImplemented
}
