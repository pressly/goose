package dialectadapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/pressly/goose/v4/internal/dialectadapter/dialectquery"
	"github.com/sethvargo/go-retry"
)

// Store is the interface that wraps the basic methods for a database dialect.
//
// A dialect is a set of SQL statements that are specific to a database.
//
// By defining a store interface, we can support multiple databases
// with a single codebase.
//
// The underlying implementation does not modify the error. It is the callers
// responsibility to assert for the correct error, such as sql.ErrNoRows.
type Store interface {
	// CreateVersionTable creates the version table within a transaction.
	// This table is used to store goose migrations.
	CreateVersionTable(ctx context.Context, tx *sql.Tx) error

	// InsertOrDelete inserts or deletes a version id from the version table within a transaction.
	InsertOrDelete(ctx context.Context, tx *sql.Tx, direction bool, version int64) error
	InsertOrDeleteNoTx(ctx context.Context, db *sql.DB, direction bool, version int64) error
	InsertOrDeleteConn(ctx context.Context, conn *sql.Conn, direction bool, version int64) error

	// GetMigration retrieves a single migration by version id.
	//
	// Returns the raw sql error if the query fails. It is the callers responsibility
	// to assert for the correct error, such as sql.ErrNoRows.
	GetMigration(ctx context.Context, conn *sql.Conn, version int64) (*GetMigrationResult, error)

	// ListMigrations retrieves all migrations sorted in descending order by id.
	//
	// If there are no migrations, an empty slice is returned with no error.
	ListMigrationsConn(ctx context.Context, conn *sql.Conn) ([]*ListMigrationsResult, error)

	// Locker defines the methods for locking the database. Some databases
	// do not support locking, in which case ErrLockNotImplemented is returned.
	Locker
}

// NewStore returns a new Store for the given dialect.
//
// The table name is used to store the goose migrations.
func NewStore(d Dialect, table string) (Store, error) {
	if table == "" {
		return nil, errors.New("table name cannot be empty")
	}
	var querier dialectquery.Querier
	switch d {
	case Postgres:
		querier = &dialectquery.Postgres{}
	case Mysql:
		querier = &dialectquery.Mysql{}
	case Sqlite3:
		querier = &dialectquery.Sqlite3{}
	case Sqlserver:
		querier = &dialectquery.Sqlserver{}
	case Redshift:
		querier = &dialectquery.Redshift{}
	case Tidb:
		querier = &dialectquery.Tidb{}
	case Clickhouse:
		querier = &dialectquery.Clickhouse{}
	case Vertica:
		querier = &dialectquery.Vertica{}
	default:
		return nil, fmt.Errorf("unknown querier dialect: %v", d)
	}
	// Retry for 60 minutes, every 2 seconds, to lock the database.
	// TODO(mf): allow users to make the duration infinite for VERY long migrations.
	retryLock := retry.WithMaxDuration(
		60*time.Minute,
		retry.NewConstant(2*time.Second),
	)
	// Retry for 1 minute, every 2 seconds, to unlock the database.
	retryUnlock := retry.WithMaxDuration(
		1*time.Minute,
		retry.NewConstant(2*time.Second),
	)
	return &store{
		tableName:   table,
		querier:     querier,
		retryLock:   retryLock,
		retryUnlock: retryUnlock,
	}, nil
}

type GetMigrationResult struct {
	IsApplied bool
	Timestamp time.Time
}

type ListMigrationsResult struct {
	Version   int64
	IsApplied bool
}

type store struct {
	tableName   string
	querier     dialectquery.Querier
	retryLock   retry.Backoff
	retryUnlock retry.Backoff
}

var _ Store = (*store)(nil)

func (s *store) CreateVersionTable(ctx context.Context, tx *sql.Tx) error {
	q := s.querier.CreateTable(s.tableName)
	_, err := tx.ExecContext(ctx, q)
	return err
}

func (s *store) InsertOrDelete(ctx context.Context, tx *sql.Tx, direction bool, version int64) error {
	if direction {
		q := s.querier.InsertVersion(s.tableName)
		_, err := tx.ExecContext(ctx, q, version, true)
		return err
	}
	q := s.querier.DeleteVersion(s.tableName)
	_, err := tx.ExecContext(ctx, q, version)
	return err
}

func (s *store) InsertOrDeleteNoTx(ctx context.Context, db *sql.DB, direction bool, version int64) error {
	if direction {
		q := s.querier.InsertVersion(s.tableName)
		_, err := db.ExecContext(ctx, q, version, true)
		return err
	}
	q := s.querier.DeleteVersion(s.tableName)
	_, err := db.ExecContext(ctx, q, version)
	return err
}

func (s *store) InsertOrDeleteConn(ctx context.Context, conn *sql.Conn, direction bool, version int64) error {
	if direction {
		q := s.querier.InsertVersion(s.tableName)
		_, err := conn.ExecContext(ctx, q, version, true)
		return err
	}
	q := s.querier.DeleteVersion(s.tableName)
	_, err := conn.ExecContext(ctx, q, version)
	return err
}

func (s *store) GetMigration(ctx context.Context, conn *sql.Conn, version int64) (*GetMigrationResult, error) {
	q := s.querier.GetMigrationByVersion(s.tableName)
	var timestamp time.Time
	var isApplied bool
	err := conn.QueryRowContext(ctx, q, version).Scan(&timestamp, &isApplied)
	if err != nil {
		return nil, err
	}
	return &GetMigrationResult{
		IsApplied: isApplied,
		Timestamp: timestamp,
	}, nil
}

func (s *store) ListMigrationsConn(ctx context.Context, conn *sql.Conn) ([]*ListMigrationsResult, error) {
	q := s.querier.ListMigrations(s.tableName)
	rows, err := conn.QueryContext(ctx, q)
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
			Version:   version,
			IsApplied: isApplied,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return migrations, nil
}
