package database

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrVersionNotFound must be returned by [GetMigration] when a migration version is not found.
	ErrVersionNotFound = errors.New("version not found")
)

// Store is an interface that defines methods for managing database migrations and versioning. By
// defining a Store interface, we can support multiple databases with consistent functionality.
//
// Each database dialect requires a specific implementation of this interface. A dialect represents
// a set of SQL statements specific to a particular database system.
type Store interface {
	// Tablename is the version table used to record applied migrations. Must not be empty.
	Tablename() string

	// CreateVersionTable creates the version table. This table is used to record applied
	// migrations. When creating the table, the implementation must also insert a row for the
	// initial version (0).
	CreateVersionTable(ctx context.Context, db DBTxConn) error

	// Insert inserts a version id into the version table.
	Insert(ctx context.Context, db DBTxConn, req InsertRequest) error

	// Delete deletes a version id from the version table.
	Delete(ctx context.Context, db DBTxConn, version int64) error

	// GetMigration retrieves a single migration by version id. If the query succeeds, but the
	// version is not found, this method must return [ErrVersionNotFound].
	GetMigration(ctx context.Context, db DBTxConn, version int64) (*GetMigrationResult, error)

	// ListMigrations retrieves all migrations sorted in descending order by id or timestamp. If
	// there are no migrations, return empty slice with no error. Typically this method will return
	// at least one migration, because the initial version (0) is always inserted into the version
	// table when it is created.
	ListMigrations(ctx context.Context, db DBTxConn) ([]*ListMigrationsResult, error)

	// TODO(mf): remove this method once the Provider is public and a custom Store can be used.
	private()
}

type InsertRequest struct {
	Version int64

	// TODO(mf): in the future, we maybe want to expand this struct so implementors can store
	// additional information. See the following issues for more information:
	//  - https://github.com/pressly/goose/issues/422
	//  - https://github.com/pressly/goose/issues/288
}

type GetMigrationResult struct {
	Timestamp time.Time
	IsApplied bool
}

type ListMigrationsResult struct {
	Version   int64
	IsApplied bool
}
