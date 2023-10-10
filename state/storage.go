package state

import (
	"context"
	"database/sql"
	"time"
)

type DB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Storage is the interface that wraps the basic methods for a database dialect.
//
// By defining a store interface, we can support multiple databases
// with a single codebase.
//
// The underlying implementation does not modify the error. It is the callers
// responsibility to assert for the correct error, such as sql.ErrNoRows.
type Storage interface {
	// CreateVersionTable creates the version table.
	// This table is used to store goose migrations.
	CreateVersionTable(ctx context.Context, db DB) error

	// InsertVersion inserts a version id into the version table.
	InsertVersion(ctx context.Context, db DB, version int64) error

	// DeleteVersion deletes a version id from the version table.
	DeleteVersion(ctx context.Context, db DB, version int64) error

	// GetMigrationRow retrieves a single migration by version id.
	//
	// Returns the raw sql error if the query fails. It is the callers responsibility
	// to assert for the correct error, such as sql.ErrNoRows.
	GetMigration(ctx context.Context, db DB, version int64) (*GetMigrationResult, error)

	// ListMigrations retrieves all migrations sorted in descending order by id.
	//
	// If there are no migrations, an empty slice is returned with no error.
	ListMigrations(ctx context.Context, db DB) ([]*ListMigrationsResult, error)
}

type GetMigrationResult struct {
	IsApplied bool
	Timestamp time.Time
}

type ListMigrationsResult struct {
	VersionID int64
	IsApplied bool
}
