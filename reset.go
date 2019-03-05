package goose

import (
	"database/sql"
	"sort"

	"github.com/pkg/errors"
)

// Reset rolls back all migrations
func Reset(db *sql.DB, dir string) error {
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return errors.Wrap(err, "failed to collect migrations")
	}
	statuses, err := dbMigrationsStatus(db)
	if err != nil {
		return errors.Wrap(err, "failed to get status of migrations")
	}
	sort.Sort(sort.Reverse(migrations))

	for _, migration := range migrations {
		if !statuses[migration.Version] {
			continue
		}
		if err = migration.Down(db); err != nil {
			return errors.Wrap(err, "failed to db-down")
		}
	}

	return nil
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
			return nil, errors.Wrap(err, "failed to scan row")
		}

		if _, ok := result[row.VersionID]; ok {
			continue
		}

		result[row.VersionID] = row.IsApplied
	}

	return result, nil
}
