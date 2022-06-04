package goose

import (
	"database/sql"
	"fmt"
	"sort"
)

// Reset rolls back all migrations
func Reset(db *sql.DB, dir string, opts ...OptionsFunc) error {
	return defaultProvider.Reset(db, dir, opts...)
}

// Reset rolls back all migrations
func (p *Provider) Reset(db *sql.DB, dir string, opts ...OptionsFunc) error {
	option := applyOptions(opts)
	migrations, err := p.CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return fmt.Errorf("failed to collect migrations: %w", err)
	}
	if option.noVersioning {
		return DownTo(db, dir, minVersion, opts...)
	}

	statuses, err := dbMigrationsStatus(p.dialect, db)
	if err != nil {
		return fmt.Errorf("failed to get status of migrations: %w", err)
	}
	sort.Sort(sort.Reverse(migrations))

	for _, migration := range migrations {
		if !statuses[migration.Version] {
			continue
		}
		if err = migration.DownWithProvider(p, db); err != nil {
			return fmt.Errorf("failed to db-down: %w", err)
		}
	}

	return nil
}

func dbMigrationsStatus(dialect SQLDialect, db *sql.DB) (map[int64]bool, error) {
	rows, err := dialect.dbVersionQuery(db)
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
