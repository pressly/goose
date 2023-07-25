package goose

import (
	"context"
	"database/sql"
)

// Locker defines the methods to lock and unlock the database.
//
// Locking is an experimental feature and the underlying implementation may change in the future.
//
// The only database that currently supports locking is Postgres. Other databases will return
// ErrLockNotImplemented.
type Locker interface {
	// LockSession and UnlockSession are used to lock the database for the duration of a session.
	//
	// The session is defined as the duration of a single connection and both methods must be called
	// on the same connection.
	LockSession(ctx context.Context, conn *sql.Conn) error
	UnlockSession(ctx context.Context, conn *sql.Conn) error

	// LockTransaction is used to lock the database for the duration of a transaction.
	// LockTransaction(ctx context.Context, tx *sql.Tx) error
}
