package dialect

import (
	"context"
	"database/sql"
)

// DBTxConn is a thin interface for common database execution methods, satisfied by
// *sql.DB, *sql.Tx, and *sql.Conn.
type DBTxConn interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// TableCreator is an optional interface that a [Querier] can implement to control
// how the version table is created, with full access to the execution context and database handle.
//
// When a [Querier] implements TableCreator, Goose delegates version table creation to
// [CreateTableContext] instead of executing the SQL query returned by [Querier.CreateTable].
// This allows dialects to introspect the database, run multiple DDL statements, handle
// DB-specific error codes, and maintain transaction awareness.
type TableCreator interface {
	CreateTableContext(ctx context.Context, db DBTxConn, tableName string) error
}
