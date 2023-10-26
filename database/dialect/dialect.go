package dialect

import (
	"context"
	"errors"
	"fmt"

	"github.com/pressly/goose/v3/database"
	"github.com/pressly/goose/v3/internal/dialect/dialectquery"
)

// Dialect is the type of database dialect.
type Dialect string

const (
	ClickHouse Dialect = "clickhouse"
	MSSQL      Dialect = "mssql"
	MySQL      Dialect = "mysql"
	Postgres   Dialect = "postgres"
	Redshift   Dialect = "redshift"
	SQLite3    Dialect = "sqlite3"
	TiDB       Dialect = "tidb"
	Vertica    Dialect = "vertica"
	YdB        Dialect = "ydb"

	// Custom is a special dialect that allows users to provide their own [database.Store]
	// implementation when constructing a [goose.Provider].
	Custom Dialect = "custom"
)

// NewStore returns a new [Store] backed by the given dialect.
func NewStore(dialect Dialect, tablename string) (database.Store, error) {
	if tablename == "" {
		return nil, errors.New("tablename must not be empty")
	}
	if dialect == "" {
		return nil, errors.New("dialect must not be empty")
	}
	if dialect == Custom {
		return nil, errors.New("dialect must not be custom")
	}
	lookup := map[Dialect]dialectquery.Querier{
		ClickHouse: &dialectquery.Clickhouse{},
		MSSQL:      &dialectquery.Sqlserver{},
		MySQL:      &dialectquery.Mysql{},
		Postgres:   &dialectquery.Postgres{},
		Redshift:   &dialectquery.Redshift{},
		SQLite3:    &dialectquery.Sqlite3{},
		TiDB:       &dialectquery.Tidb{},
		Vertica:    &dialectquery.Vertica{},
	}
	querier, ok := lookup[dialect]
	if !ok {
		return nil, fmt.Errorf("unknown dialect: %q", dialect)
	}
	return &store{
		tablename: tablename,
		querier:   querier,
	}, nil
}

type store struct {
	tablename string
	querier   dialectquery.Querier
}

var _ database.Store = (*store)(nil)

func (s *store) CreateVersionTable(ctx context.Context, db database.DBTxConn) error {
	q := s.querier.CreateTable(s.tablename)
	if _, err := db.ExecContext(ctx, q); err != nil {
		return fmt.Errorf("failed to create version table %q: %w", s.tablename, err)
	}
	return nil
}

func (s *store) InsertOrDelete(ctx context.Context, db database.DBTxConn, direction bool, version int64) error {
	if direction {
		q := s.querier.InsertVersion(s.tablename)
		if _, err := db.ExecContext(ctx, q, version, true); err != nil {
			return fmt.Errorf("failed to insert version %d: %w", version, err)
		}
		return nil
	}
	q := s.querier.DeleteVersion(s.tablename)
	if _, err := db.ExecContext(ctx, q, version); err != nil {
		return fmt.Errorf("failed to delete version %d: %w", version, err)
	}
	return nil
}

func (s *store) GetMigration(
	ctx context.Context,
	db database.DBTxConn,
	version int64) (*database.GetMigrationResult, error) {
	q := s.querier.GetMigrationByVersion(s.tablename)
	var result database.GetMigrationResult
	if err := db.QueryRowContext(ctx, q, version).Scan(
		&result.Timestamp,
		&result.IsApplied,
	); err != nil {
		return nil, fmt.Errorf("failed to get migration %d: %w", version, err)
	}
	return &result, nil
}

func (s *store) ListMigrations(
	ctx context.Context,
	db database.DBTxConn,
) ([]*database.ListMigrationsResult, error) {
	q := s.querier.ListMigrations(s.tablename)
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("failed to list migrations: %w", err)
	}
	defer rows.Close()

	var migrations []*database.ListMigrationsResult
	for rows.Next() {
		var result database.ListMigrationsResult
		if err := rows.Scan(&result.Version, &result.IsApplied); err != nil {
			return nil, fmt.Errorf("failed to scan list migrations result: %w", err)
		}
		migrations = append(migrations, &result)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return migrations, nil
}
