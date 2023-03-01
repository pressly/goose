package dialect

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/pressly/goose/v3/internal/dialect/dialectquery"
)

type Dialect string

const (
	Postgres   Dialect = "postgres"
	Mysql      Dialect = "mysql"
	Sqlite3    Dialect = "sqlite3"
	Sqlserver  Dialect = "sqlserver"
	Redshift   Dialect = "redshift"
	Tidb       Dialect = "tidb"
	Clickhouse Dialect = "clickhouse"
	Vertica    Dialect = "vertica"
)

func NewDialectStore(d Dialect, table string) (DialectStore, error) {
	if table == "" {
		return nil, errors.New("table name cannot be empty")
	}
	var querier dialectquery.Querier
	switch d {
	case Postgres:
		querier = dialectquery.NewPostgres(table)
	case Mysql:
		querier = dialectquery.NewMysql(table)
	case Sqlite3:
		querier = dialectquery.NewSqlite3(table)
	case Sqlserver:
		querier = dialectquery.NewSqlserver(table)
	case Redshift:
		querier = dialectquery.NewRedshift(table)
	case Tidb:
		querier = dialectquery.NewTidb(table)
	case Clickhouse:
		querier = dialectquery.NewClickhouse(table)
	case Vertica:
		querier = dialectquery.NewVertica(table)
	default:
		return nil, errors.New("unknown dialect")
	}
	return &store{querier: querier}, nil
}

// DialectStore is the interface that wraps the basic methods to create a
// dialect specific query.
//
// A dialect is a set of SQL statements that are specific to a database.
//
// By defining a dialect store interface, we can support multiple databases
// with a single codebase.
//
// The underlying implementation does not modify the error returned by the
// database driver. It is the callers responsibility to assert for the correct
// error, such as sql.ErrNoRows.
type DialectStore interface {
	// CreateVersionTable creates the version table within a transaction.
	// This table is used to store goose migrations.
	CreateVersionTable(ctx context.Context, tx *sql.Tx) error

	// InsertVersion inserts a version id into the version table within a transaction.
	InsertVersion(ctx context.Context, tx *sql.Tx, version int64) error
	// InsertVersionNoTx inserts a version id into the version table without a transaction.
	InsertVersionNoTx(ctx context.Context, db *sql.DB, version int64) error

	// DeleteVersion deletes a version id from the version table within a transaction.
	DeleteVersion(ctx context.Context, tx *sql.Tx, version int64) error
	// DeleteVersionNoTx deletes a version id from the version table without a transaction.
	DeleteVersionNoTx(ctx context.Context, db *sql.DB, version int64) error

	// TODO(mf): this is inefficient. Only used in one place to list migrations one-by-one
	// but we can do better. Oh, and selecting by version id does not have an index ...
	//
	// GetMigrationRow retrieves a single migration by version id.
	//
	// Returns the raw sql error if the query fails. It is the callers responsibility
	// to assert for the correct error, such as sql.ErrNoRows.
	GetMigration(ctx context.Context, db *sql.DB, version int64) (*MigrationRow, error)

	// ListMigrations retrieves all migrations sorted in descending order.
	// If there are no migrations, an empty slice is returned with no error.
	//
	// Note, the *MigrationRow object does not have a timestamp field.
	ListMigrations(ctx context.Context, db *sql.DB) ([]*MigrationRow, error)
}

type MigrationRow struct {
	VersionID int64
	IsApplied bool
	Timestamp time.Time
}
