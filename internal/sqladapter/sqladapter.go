// Package sqladapter provides an interface for interacting with a SQL database.
//
// All supported database dialects must implement the Store interface.
package sqladapter

import (
	"context"
	"database/sql"
	"time"
)

// DBTxConn is an interface that is satisfied by *sql.DB, *sql.Tx and *sql.Conn.
//
// There is a long outstanding issue to formalize a std lib interface, but alas...
// See: https://github.com/golang/go/issues/14468
type DBTxConn interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// Store is the interface that wraps the basic methods for a database dialect.
//
// A dialect is a set of SQL statements that are specific to a database.
//
// By defining a store interface, we can support multiple databases with a single codebase.
//
// The underlying implementation does not modify the error. It is the callers responsibility to
// assert for the correct error, such as sql.ErrNoRows.
type Store interface {
	// CreateVersionTable creates the version table within a transaction. This table is used to
	// record applied migrations.
	CreateVersionTable(ctx context.Context, tx *sql.Tx, tablename string) error

	// InsertOrDelete inserts or deletes a version id from the version table.
	InsertOrDelete(ctx context.Context, db DBTxConn, direction bool, version int64) error

	// GetMigration retrieves a single migration by version id.
	//
	// Returns the raw sql error if the query fails. It is the callers responsibility
	// to assert for the correct error, such as sql.ErrNoRows.
	GetMigrationConn(ctx context.Context, conn *sql.Conn, version int64) (*GetMigrationResult, error)

	// ListMigrations retrieves all migrations sorted in descending order by id.
	//
	// If there are no migrations, an empty slice is returned with no error.
	ListMigrationsConn(ctx context.Context, conn *sql.Conn) ([]*ListMigrationsResult, error)
}

type GetMigrationResult struct {
	IsApplied bool
	Timestamp time.Time
}

type ListMigrationsResult struct {
	Version   int64
	IsApplied bool
}
