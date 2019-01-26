package goose

import (
	"database/sql"
	"fmt"
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
	for {
		current, err := GetDBVersion(db)
		if err != nil {
			return err
		}

		next, err := migrations.Next(current)
		if err != nil {
			if err == ErrNoNextVersion {
				log.Printf("goose: no migrations to run. current version: %d\n", current)
				return nil
			}
			return err
		}
		if endVersion != nil && next.Version > *endVersion {
			break
		}
		if err = next.Up(db); err != nil {
			return err
		}
		if onlyOne {
			break
		}
	}
	return fmt.Errorf("should not happen")
}
