package database

import (
	"context"
	"github.com/pressly/goose/v4/migration"

	"github.com/pressly/goose/v4/internal/dialectstore"
)

// Store is an interface that defines methods for tracking and managing migrations. It is used by
// the goose package to interact with a database. By defining a Store interface, multiple
// implementations can be created to support different databases without reimplementing the
// migration logic.
//
// This package provides several dialects that implement the Store interface. While most users won't
// need to create their own Store, if you need to support a database that isn't currently supported,
// you can implement your own!
type Store interface {
	// Tablename is the name of the version table. This table is used to record applied migrations
	// and must not be an empty string.
	Tablename() string
	// CreateVersionTable creates the version table, which is used to track migrations.
	CreateVersionTable(ctx context.Context, db DBTxConn) error
	// Insert a version id into the version table.
	Insert(ctx context.Context, db DBTxConn, req migration.Entity) error
	// Delete a version id from the version table.
	Delete(ctx context.Context, db DBTxConn, version int64) error
	// GetMigration retrieves a single migration by version id. If the query succeeds, but the
	// version is not found, this method must return [ErrVersionNotFound].
	GetMigration(ctx context.Context, db DBTxConn, version int64) (*dialectstore.GetMigrationResult, error)
	// GetLatestVersion retrieves the last applied migration version. If no migrations exist, this
	// method must return [ErrVersionNotFound].
	GetLatestVersion(ctx context.Context, db DBTxConn) (int64, error)
	// ListMigrations retrieves all migrations sorted in descending order by id or timestamp. If
	// there are no migrations, return empty slice with no error. Typically this method will return
	// at least one migration, because the initial version (0) is always inserted into the version
	// table when it is created.
	ListMigrations(ctx context.Context, db DBTxConn) ([]*dialectstore.ListMigrationsResult, error)
}

type InsertRequest struct {
	Version int64
}
