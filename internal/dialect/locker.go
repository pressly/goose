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
			if _, err := conn.ExecContext(ctx, t.AdvisoryUnlockSession(), defaultLockID); err != nil {
				return retry.RetryableError(err)
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
