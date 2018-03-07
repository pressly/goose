package goose

import (
	"database/sql"
	"log"
	"sort"
)

// Reset rolls back all migrations
func Reset(db *sql.DB, schemaID, dir string) error {
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}
	statuses, err := dbMigrationsStatus(db, schemaID)
	if err != nil {
		return err
	}
	sort.Sort(sort.Reverse(migrations))

	for _, migration := range migrations {
		if !statuses[migration.Version] {
			continue
		}
		if err = migration.Down(db, schemaID); err != nil {
			return err
		}
	}

	return nil
}

func dbMigrationsStatus(db *sql.DB, schemaID string) (map[int64]bool, error) {
	rows, err := GetDialect().dbVersionQuery(db, schemaID)
	if err != nil {
		return map[int64]bool{}, createVersionTable(db, schemaID)
	}
	defer rows.Close()

	// The most recent record for each migration specifies
	// whether it has been applied or rolled back.

	result := map[int64]bool{
		0: true,
	}

	for rows.Next() {
		var row MigrationRecord
		if err = rows.Scan(&row.VersionID, &row.IsApplied); err != nil {
			log.Fatal("error scanning rows:", err)
		}

		if _, ok := result[row.VersionID]; ok {
			continue
		}

		result[row.VersionID] = row.IsApplied
	}

	return result, nil
}
