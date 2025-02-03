package dialectstore

import (
	"context"
	"errors"
	"fmt"
	"github.com/pressly/goose/v3/internal/dialect"
	"github.com/pressly/goose/v3/internal/dialectquery"
	"github.com/pressly/goose/v3/internal/sql"
	"github.com/pressly/goose/v3/migration"
	"time"
)

var (
	// ErrVersionNotFound must be returned by [GetMigration] or [GetLatestVersion] when a migration
	// does not exist.
	ErrVersionNotFound = errors.New("version not found")
)

// Store is the interface that wraps the basic methods for a database dialect.
//
// A dialect is a set of SQL statements that are specific to a database.
//
// By defining a store interface, we can support multiple databases with a single codebase.
//
// The underlying implementation does not modify the error. It is the callers
// responsibility to assert for the correct error, such as sql.ErrNoRows.
type Store interface {
	migration.StoreVersionTable

	GetDialect() dialect.Dialect

	// InsertVersion inserts a version id into the version table within a transaction.
	InsertVersion(ctx context.Context, tx sql.DBTxConn, entity migration.Entity) error
	// InsertVersionNoTx inserts a version id into the version table without a transaction.
	InsertVersionNoTx(ctx context.Context, db sql.DBTxConn, version migration.Entity) error

	// DeleteVersion deletes a version id from the version table within a transaction.
	DeleteVersion(ctx context.Context, tx sql.DBTxConn, entity migration.Entity) error
	// DeleteVersionNoTx deletes a version id from the version table without a transaction.
	DeleteVersionNoTx(ctx context.Context, db sql.DBTxConn, entity migration.Entity) error

	// GetLatestVersion retrieves the last applied migration version. If no migrations exist, this
	// method must return [ErrVersionNotFound].
	GetLatestVersion(ctx context.Context, db sql.DBTxConn) (migration.VersionID, error)

	// GetMigrationRow retrieves a single migration by version id.
	//
	// Returns the raw sql error if the query fails. It is the callers responsibility
	// to assert for the correct error, such as sql.ErrNoRows.
	GetMigration(ctx context.Context, db sql.DBTxConn, version migration.VersionID) (*GetMigrationResult, error)
	// ListMigrations retrieves all migrations sorted in descending order by id.
	//
	// If there are no migrations, an empty slice is returned with no error.
	ListMigrations(ctx context.Context, db sql.DBTxConn) ([]*ListMigrationsResult, error)
}

type GetMigrationResult struct {
	IsApplied bool
	Timestamp time.Time
}

type ListMigrationsResult struct {
	VersionID int64
	IsApplied bool
}

var _ Store = (*store)(nil)

// NewStore returns a new Store for the given dialect.
func NewStore(d dialect.Dialect, tableName string) (Store, error) {
	if tableName == "" {
		return nil, errors.New("table name must not be empty")
	}

	var querier, err = dialectquery.LookupQuerier(d)
	if err != nil {
		return nil, err
	}

	return &store{querier: querier, tableName: tableName}, nil
}

type store struct {
	tableName string
	querier   dialectquery.Querier
}

func (s *store) GetTableName() string { return s.tableName }

func (s *store) CreateVersionTable(ctx context.Context, tx sql.DBTxConn) error {
	q := s.querier.CreateTable(s.tableName)

	if _, err := tx.ExecContext(ctx, q); err != nil {
		return fmt.Errorf("failed to create version table %q: %w", s.tableName, err)
	}

	return nil
}

func (s *store) TableVersionExists(ctx context.Context, tx sql.DBTxConn) (bool, error) {
	q := s.querier.TableExists(s.tableName)
	if q == "" {
		return false, errors.ErrUnsupported
	}

	var exists bool
	// Note, we do not pass the table name as an argument to the query, as the query should be
	// pre-defined by the dialect.
	if err := tx.QueryRowContext(ctx, q).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check if table exists: %w", err)
	}

	return exists, nil
}

func (s *store) GetDialect() dialect.Dialect { return s.querier.GetDialect() }

func (s *store) InsertVersion(ctx context.Context, tx sql.DBTxConn, entity migration.Entity) error {
	q := s.querier.InsertVersion(s.tableName)
	if _, err := tx.ExecContext(ctx, q, entity.Version, true); err != nil {
		return fmt.Errorf("failed to insert version %d: %w", entity.Version, err)
	}

	return nil
}

func (s *store) InsertVersionNoTx(ctx context.Context, db sql.DBTxConn, entity migration.Entity) error {
	q := s.querier.InsertVersion(s.tableName)
	_, err := db.ExecContext(ctx, q, entity.Version, true)
	return err
}

func (s *store) DeleteVersion(ctx context.Context, tx sql.DBTxConn, entity migration.Entity) error {
	q := s.querier.DeleteVersion(s.tableName)
	if _, err := tx.ExecContext(ctx, q, entity.Version); err != nil {
		return fmt.Errorf("failed to delete version %d: %w", entity.Version, err)
	}

	return nil
}

func (s *store) DeleteVersionNoTx(ctx context.Context, db sql.DBTxConn, entity migration.Entity) error {
	q := s.querier.DeleteVersion(s.tableName)
	_, err := db.ExecContext(ctx, q, entity.Version)
	return err
}

func (s *store) GetLatestVersion(ctx context.Context, db sql.DBTxConn) (migration.VersionID, error) {
	q := s.querier.GetLatestVersion(s.tableName)

	var version sql.NullInt64
	err := db.QueryRowContext(ctx, q).Scan(&version)

	if err != nil {
		return -1, fmt.Errorf("failed to get latest version: %w", err)
	}

	if !version.Valid {
		return -1, fmt.Errorf("latest %w", ErrVersionNotFound)
	}

	return version.Int64, nil
}

func (s *store) GetMigration(ctx context.Context, db sql.DBTxConn, version int64) (*GetMigrationResult, error) {
	q := s.querier.GetMigrationByVersion(s.tableName)

	var result GetMigrationResult

	err := db.QueryRowContext(ctx, q, version).Scan(&result.Timestamp, &result.IsApplied)
	if err == nil {
		return &result, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %d", ErrVersionNotFound, version)
	}

	return nil, fmt.Errorf("failed to get migration %d: %w", version, err)
}

func (s *store) ListMigrations(ctx context.Context, db sql.DBTxConn) ([]*ListMigrationsResult, error) {
	q := s.querier.ListMigrations(s.tableName)
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []*ListMigrationsResult
	for rows.Next() {
		var result ListMigrationsResult
		if err := rows.Scan(&result.VersionID, &result.IsApplied); err != nil {
			return nil, err
		}
		migrations = append(migrations, &result)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return migrations, nil
}
