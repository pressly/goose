package dialect

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/pressly/goose/v3/internal/dialect/dialectquery"
)

// Dialect is the type of database dialect.
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

func NewStore(d Dialect, table string) (Store, error) {
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
		return nil, fmt.Errorf("unknown querier dialect: %v", d)
	}
	return &store{querier: querier}, nil
}

// Store is the interface that wraps the basic methods for a database dialect.
//
// A dialect is a set of SQL statements that are specific to a database.
//
// By defining a store interface, we can support multiple databases
// with a single codebase.
//
// The underlying implementation does not modify the error returned by the
// database driver. It is the callers responsibility to assert for the correct
// error, such as sql.ErrNoRows.
type Store interface {
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
	GetMigration(ctx context.Context, db *sql.DB, version int64) (*GetMigrationResult, error)

	// ListMigrations retrieves all migrations sorted in descending order by id.
	// If there are no migrations, an empty slice is returned with no error.
	//
	// Note, the *MigrationRow object does not have a timestamp field.
	ListMigrations(ctx context.Context, db *sql.DB) ([]*ListMigrationsResult, error)
}

type GetMigrationResult struct {
	IsApplied bool
	Timestamp time.Time
}

type ListMigrationsResult struct {
	VersionID int64
	IsApplied bool
}

type store struct {
	querier dialectquery.Querier
}

var _ Store = (*store)(nil)

func (s *store) CreateVersionTable(ctx context.Context, tx *sql.Tx) error {
	q := s.querier.CreateTable()
	_, err := tx.ExecContext(ctx, q)
	return err
}

func (s *store) InsertVersion(ctx context.Context, tx *sql.Tx, version int64) error {
	q := s.querier.InsertVersion()
	_, err := tx.ExecContext(ctx, q, version, true)
	return err
}

func (s *store) InsertVersionNoTx(ctx context.Context, db *sql.DB, version int64) error {
	q := s.querier.InsertVersion()
	_, err := db.ExecContext(ctx, q, version, true)
	return err
}

func (s *store) DeleteVersion(ctx context.Context, tx *sql.Tx, version int64) error {
	q := s.querier.DeleteVersion()
	_, err := tx.ExecContext(ctx, q, version)
	return err
}

func (s *store) DeleteVersionNoTx(ctx context.Context, db *sql.DB, version int64) error {
	q := s.querier.DeleteVersion()
	_, err := db.ExecContext(ctx, q, version)
	return err
}

func (s *store) GetMigration(ctx context.Context, db *sql.DB, version int64) (*GetMigrationResult, error) {
	q := s.querier.GetMigrationByVersion()
	var timestamp time.Time
	var isApplied bool
	err := db.QueryRowContext(ctx, q, version).Scan(&timestamp, &isApplied)
	if err != nil {
		return nil, err
	}
	return &GetMigrationResult{
		IsApplied: isApplied,
		Timestamp: timestamp,
	}, nil
}

func (s *store) ListMigrations(ctx context.Context, db *sql.DB) ([]*ListMigrationsResult, error) {
	q := s.querier.ListMigrations()
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []*ListMigrationsResult
	for rows.Next() {
		var version int64
		var isApplied bool
		if err := rows.Scan(&version, &isApplied); err != nil {
			return nil, err
		}
		migrations = append(migrations, &ListMigrationsResult{
			VersionID: version,
			IsApplied: isApplied,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return migrations, nil
}
