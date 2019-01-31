package goose

import (
	"database/sql"
)

// Up performs all types of migrations upwards, depending on the params.
func Up(db *sql.DB, dir string, includeMissing bool, onlyOne bool, endVersion *int64) error {
	var migrations Migrations
	currentVersion, err := GetDBVersion(db)
	if err != nil {
		return err
	}
	if includeMissing {
		var err error
		migrations, err = MissingMigrations(db, dir)
		if err != nil {
			return err
		}
	} else {
		migrations, err = CollectMigrations(dir, currentVersion, maxVersion)
		if err != nil {
			return err
		}
	}
	for _, migration := range migrations {
		if endVersion != nil && migration.Version > *endVersion {
			break
		}
		// Only update version number if we are applying a missed migration.
		updateVersion := migration.Version > currentVersion
		if err := migration.Up(db, updateVersion); err != nil {
			return err
		}
		if onlyOne {
			break
		}
	}
	log.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
	return nil
}
