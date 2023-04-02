package dialectadapter

import (
	"context"
	"database/sql"
	"errors"
	"hash/crc64"

	"github.com/pressly/goose/v4/internal/dialectadapter/dialectquery"
	"github.com/sethvargo/go-retry"
)

var (
	// defaultLockID is the id used to lock the database for migrations. It is a crc64 hash of the
	// string "goose". This is used to ensure that the lock is unique to goose.
	//
	// 5887940537704921958
	defaultLockID = crc64.Checksum([]byte("goose"), crc64.MakeTable(crc64.ECMA))

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
	// IsSupported returns true if the database supports locking.
	IsSupported() bool

	// LockSession and UnlockSession are used to lock the database for the duration of a session.
	//
	// The session is defined as the duration of a single connection.
	LockSession(ctx context.Context, conn *sql.Conn) error
	UnlockSession(ctx context.Context, conn *sql.Conn) error

	// LockTransaction is used to lock the database for the duration of a transaction.
	LockTransaction(ctx context.Context, tx *sql.Tx) error
}

func (s *store) IsSupported() bool {
	switch s.querier.(type) {
	case *dialectquery.Postgres:
		return true
	default:
		return false
	}
}

func (s *store) LockSession(ctx context.Context, conn *sql.Conn) error {
	switch t := s.querier.(type) {
	case *dialectquery.Postgres:
		// TODO(mf): need to be VERY careful about the retry logic here to avoid stacking locks on
		// top of each other. We need to make sure that we only retry if the lock is not already
		// held.
		//
		// This retry is a bit pointless because if we can't get the lock, chances are another
		// process is holding the lock and we will just spin here forever. We should probably just
		// remove this retry.
		//
		// At best this might help with a transient network issue.
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

				// TODO(mf): provide the user with some documentation on how they can unlock the
				// session manually. Although this is probably an issue for 99.9% of users
				// since pg_advisory_unlock_all() will release all session level advisory locks held
				// by the current session. (This function is implicitly invoked at session end, even
				// if the client disconnects ungracefully.)
				//
				// TODO(mf): - we may not want to bother checking the return value and just assume
				// that the lock was released. This would simplify the code and remove the need for
				// the unlocked bool.
				//
				// SELECT pid,granted,((classid::bigint<<32)|objid::bigint)AS goose_lock_id FROM
				// pg_locks WHERE locktype='advisory';
				//
				// | pid | granted | goose_lock_id       |
				// |-----|---------|---------------------|
				// | 191 | t       | 5887940537704921958 |
				//
				// A more forceful way to unlock the session is to terminate the process: SELECT
				// pg_terminate_backend(120);

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
