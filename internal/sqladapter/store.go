package sqladapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/pressly/goose/v3/internal/dialect/dialectquery"
	"github.com/pressly/goose/v3/internal/sqlextended"
)

var _ Store = (*store)(nil)

type store struct {
	tablename string
	querier   dialectquery.Querier
}

// NewStore returns a new Store backed by the given dialect.
//
// The dialect must match one of the supported dialects defined in dialect.go.
func NewStore(dialect string, table string) (Store, error) {
	if table == "" {
		return nil, errors.New("table must not be empty")
	}
	if dialect == "" {
		return nil, errors.New("dialect must not be empty")
	}
	var querier dialectquery.Querier
	switch dialect {
	case "clickhouse":
		querier = &dialectquery.Clickhouse{}
	case "mssql":
		querier = &dialectquery.Sqlserver{}
	case "mysql":
		querier = &dialectquery.Mysql{}
	case "postgres":
		querier = &dialectquery.Postgres{}
	case "redshift":
		querier = &dialectquery.Redshift{}
	case "sqlite3":
		querier = &dialectquery.Sqlite3{}
	case "tidb":
		querier = &dialectquery.Tidb{}
	case "vertica":
		querier = &dialectquery.Vertica{}
	default:
		return nil, fmt.Errorf("unknown dialect: %q", dialect)
	}
	return &store{
		tablename: table,
		querier:   querier,
	}, nil
}

func (s *store) CreateVersionTable(ctx context.Context, db sqlextended.DBTxConn) error {
	q := s.querier.CreateTable(s.tablename)
	if _, err := db.ExecContext(ctx, q); err != nil {
		return fmt.Errorf("failed to create version table %q: %w", s.tablename, err)
	}
	return nil
}

func (s *store) InsertOrDelete(ctx context.Context, db sqlextended.DBTxConn, direction bool, version int64) error {
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

func (s *store) GetMigration(ctx context.Context, db sqlextended.DBTxConn, version int64) (*GetMigrationResult, error) {
	q := s.querier.GetMigrationByVersion(s.tablename)
	result := new(GetMigrationResult)
	if err := db.QueryRowContext(ctx, q, version).Scan(
		&result.Timestamp,
		&result.IsApplied,
	); err != nil {
		return nil, fmt.Errorf("failed to get migration %d: %w", version, err)
	}
	return result, nil
}

func (s *store) ListMigrations(ctx context.Context, db sqlextended.DBTxConn) ([]*ListMigrationsResult, error) {
	q := s.querier.ListMigrations(s.tablename)
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("failed to list migrations: %w", err)
	}
	defer rows.Close()

	var migrations []*ListMigrationsResult
	for rows.Next() {
		result := new(ListMigrationsResult)
		if err := rows.Scan(&result.Version, &result.IsApplied); err != nil {
			return nil, fmt.Errorf("failed to scan list migrations result: %w", err)
		}
		migrations = append(migrations, result)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return migrations, nil
}
