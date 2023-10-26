package database

import (
	"context"
	"time"
)

// Store is an interface that defines methods for managing database migrations and versioning. By
// defining a Store interface, we can support multiple databases with consistent functionality.
//
// Each database dialect requires a specific implementation of this interface. A dialect represents
// a set of SQL statements specific to a particular database system.
type Store interface {
	// CreateVersionTable creates the version table. This table is used to record applied
	// migrations.
	CreateVersionTable(ctx context.Context, db DBTxConn) error

	// InsertOrDelete inserts or deletes a version id from the version table. If direction is true,
	// insert the version id. If direction is false, delete the version id.
	InsertOrDelete(ctx context.Context, db DBTxConn, direction bool, version int64) error

	// GetMigration retrieves a single migration by version id. This method may return the raw sql
	// error if the query fails so the caller can assert for errors such as [sql.ErrNoRows].
	GetMigration(ctx context.Context, db DBTxConn, version int64) (*GetMigrationResult, error)

	// ListMigrations retrieves all migrations sorted in descending order by id or timestamp. If
	// there are no migrations, return empty slice with no error.
	ListMigrations(ctx context.Context, db DBTxConn) ([]*ListMigrationsResult, error)
}

type GetMigrationResult struct {
	Timestamp time.Time
	IsApplied bool
}

type ListMigrationsResult struct {
	Version   int64
	IsApplied bool
}
