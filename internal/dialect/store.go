package dialect

import (
	"context"
	"database/sql"
	"time"

	"github.com/pressly/goose/v3/internal/dialect/dialectquery"
)

type store struct {
	querier dialectquery.Querier
}

var _ DialectStore = (*store)(nil)

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

func (s *store) GetMigration(ctx context.Context, db *sql.DB, version int64) (*MigrationRow, error) {
	q := s.querier.GetMigrationByVersion()
	var timestamp time.Time
	var isApplied bool
	err := db.QueryRowContext(ctx, q, version).Scan(&timestamp, &isApplied)
	if err != nil {
		return nil, err
	}
	return &MigrationRow{
		VersionID: version,
		IsApplied: isApplied,
		Timestamp: timestamp,
	}, nil
}

func (s *store) ListMigrations(ctx context.Context, db *sql.DB) ([]*MigrationRow, error) {
	q := s.querier.ListMigrations()
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []*MigrationRow
	for rows.Next() {
		var version int64
		var isApplied bool
		if err := rows.Scan(&version, &isApplied); err != nil {
			return nil, err
		}
		migrations = append(migrations, &MigrationRow{
			VersionID: version,
			IsApplied: isApplied,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return nil, nil
}
