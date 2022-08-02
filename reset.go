package goose

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
)

// ResetCtx rolls back all migrations
func ResetCtx(ctx context.Context, db *sql.DB, dir string, opts ...OptionsFunc) error {
	option := &options{}
	for _, f := range opts {
		f(option)
	}
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return fmt.Errorf("failed to collect migrations: %w", err)
	}
	if option.noVersioning {
		return DownTo(db, dir, minVersion, opts...)
	}

	statuses, err := dbMigrationsStatus(db)
	if err != nil {
		return fmt.Errorf("failed to get status of migrations: %w", err)
	}
	sort.Sort(sort.Reverse(migrations))

	for _, migration := range migrations {
		if !statuses[migration.Version] {
			continue
		}
		if err = migration.DownCtx(ctx, db); err != nil {
			return fmt.Errorf("failed to db-down: %w", err)
		}
	}

	return nil
}

// Reset rolls back all migrations
//
// Reset uses context.Background internally; to specify the context, use ResetCtx.
func Reset(db *sql.DB, dir string, opts ...OptionsFunc) error {
	return ResetCtx(context.Background(), db, dir, opts...)
}

func dbMigrationsStatus(db *sql.DB) (map[int64]bool, error) {
	rows, err := GetDialect().dbVersionQuery(db)
	if err != nil {
		return map[int64]bool{}, nil
	}
	defer rows.Close()

	// The most recent record for each migration specifies
	// whether it has been applied or rolled back.

	result := make(map[int64]bool)

	for rows.Next() {
		var row MigrationRecord
		if err = rows.Scan(&row.VersionID, &row.IsApplied); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		if _, ok := result[row.VersionID]; ok {
			continue
		}

		result[row.VersionID] = row.IsApplied
	}

	return result, nil
}
