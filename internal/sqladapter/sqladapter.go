// Package sqladapter provides an interface for interacting with a SQL database.
//
// All supported database dialects must implement the Store interface.
package sqladapter

import (
	"context"
	"time"

	"github.com/pressly/goose/v3/internal/sqlextended"
)

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
	CreateVersionTable(ctx context.Context, db sqlextended.DBTxConn) error

	// InsertOrDelete inserts or deletes a version id from the version table.
	InsertOrDelete(ctx context.Context, db sqlextended.DBTxConn, direction bool, version int64) error

	// GetMigration retrieves a single migration by version id.
	//
	// Returns the raw sql error if the query fails. It is the callers responsibility
	// to assert for the correct error, such as sql.ErrNoRows.
	GetMigration(ctx context.Context, db sqlextended.DBTxConn, version int64) (*GetMigrationResult, error)

	// ListMigrations retrieves all migrations sorted in descending order by id.
	//
	// If there are no migrations, an empty slice is returned with no error.
	ListMigrations(ctx context.Context, db sqlextended.DBTxConn) ([]*ListMigrationsResult, error)
}

type GetMigrationResult struct {
	IsApplied bool
	Timestamp time.Time
}

type ListMigrationsResult struct {
	Version   int64
	IsApplied bool
}
