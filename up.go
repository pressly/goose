package goose

import (
	"database/sql"
)

// Up performs all types of migrations upwards, depending on the params.
func Up(db *sql.DB, dir string, includeMissing bool, onlyOne bool, endVersion *int64) error {
	var migrations Migrations
	if includeMissing {
		var err error
		migrations, err = MissingMigrations(db, dir)
		if err != nil {
			return err
		}
	} else {
		current, err := GetDBVersion(db)
		if err != nil {
			return err
		}
		migrations, err = CollectMigrations(dir, current, maxVersion)
		if err != nil {
			return err
		}
	}
	for _, migration := range migrations {
		if endVersion != nil && migration.Version > *endVersion {
			break
		}
		if err := migration.Up(db); err != nil {
			return err
		}
		if onlyOne {
			break
		}
	}
	currentVersion, err := GetDBVersion(db)
	if err != nil {
		return err
	}
	if includeMissing {
		// The version has been set based on the last migration we just applied.
		// However, since we included missing migrations, that migration might
		// not be the latest one.
		actualCurrentVersion, err := GetLatestAppliedMigrationVersion(db, dir)
		if err != nil {
			return err
		}
		if actualCurrentVersion != currentVersion {
			err := SetDBVersion(db, actualCurrentVersion)
			if err != nil {
				return err
			}
			currentVersion = actualCurrentVersion
		}
	}
	log.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
	return nil
}
