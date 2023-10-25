package storage

import (
	"context"
	"time"

	"github.com/pressly/goose/v3/state"
)

const defaultTablename = "goose_db_version"

type queries struct {
	createTable           string
	insertVersion         string
	deleteVersion         string
	getMigrationByVersion string
	listMigrations        string
}

// CreateVersionTable implements Store.
func (q queries) CreateVersionTable(ctx context.Context, db state.DB) error {
	_, err := db.ExecContext(ctx, q.createTable)
	return err
}

// InsertVersion implements Store.
func (q queries) InsertVersion(ctx context.Context, db state.DB, version int64) error {
	_, err := db.ExecContext(ctx, q.insertVersion, version, true)
	return err
}

// DeleteVersion implements Store.
func (q queries) DeleteVersion(ctx context.Context, db state.DB, version int64) error {
	_, err := db.ExecContext(ctx, q.deleteVersion, version)
	return err
}

// GetMigration implements Store.
func (q queries) GetMigration(ctx context.Context, db state.DB, version int64) (*state.GetMigrationResult, error) {
	var timestamp time.Time
	var isApplied bool
	err := db.QueryRowContext(ctx, q.getMigrationByVersion, version).Scan(&timestamp, &isApplied)
	if err != nil {
		return nil, err
	}
	return &state.GetMigrationResult{
		IsApplied: isApplied,
		Timestamp: timestamp,
	}, nil
}

// ListMigrations implements Store.
func (q queries) ListMigrations(ctx context.Context, db state.DB) ([]*state.ListMigrationsResult, error) {
	rows, err := db.QueryContext(ctx, q.listMigrations)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []*state.ListMigrationsResult
	for rows.Next() {
		var version int64
		var isApplied bool
		if err := rows.Scan(&version, &isApplied); err != nil {
			return nil, err
		}
		migrations = append(migrations, &state.ListMigrationsResult{
			VersionID: version,
			IsApplied: isApplied,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return migrations, nil
}
