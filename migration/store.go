package migration

import (
	"context"
	"github.com/pressly/goose/v4/internal/sql"
)

type StoreVersionTable interface {
	// GetTableName is the name of the version table. This table is used to record applied migrations
	// and must not be an empty string.
	GetTableName() string

	// CreateVersionTable creates the version table within a transaction.
	// This table is used to store goose migrations.
	CreateVersionTable(ctx context.Context, tx sql.DBTxConn) error

	// TableVersionExists checks if the migrations table exists in the database. Implementing this method
	// allows goose to optimize table existence checks by using database-specific system catalogs
	// (e.g., pg_tables for PostgreSQL, sqlite_master for SQLite) instead of generic SQL queries.
	//
	// Return [errors.ErrUnsupported] if the database does not provide an efficient way to check
	// table existence.
	TableVersionExists(ctx context.Context, tx sql.DBTxConn) (bool, error)
}
