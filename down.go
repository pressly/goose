package goose

import (
	"database/sql"
	"fmt"
)

// Down rolls back a single migration from the current version.
func (ms Migrations) Down(db *sql.DB) error {
	currentVersion, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	current, err := ms.Current(currentVersion)
	if err != nil {
		return fmt.Errorf("no migration %v", currentVersion)
	}

	return current.Down(db)
}

// DownTo rolls back migrations to a specific version.
func (ms Migrations) DownTo(db *sql.DB, targetVersion int64) error {
	for {
		currentVersion, err := GetDBVersion(db)
		if err != nil {
			return err
		}

		current, err := ms.Current(currentVersion)
		if err != nil || currentVersion <= targetVersion {
			log.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
			return nil
		}

		if err = current.Down(db); err != nil {
			return err
		}
	}
}

// Down rolls back a single migration from the current version.
func Down(db *sql.DB, dir string) error {
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	return migrations.Down(db)
}

// DownTo rolls back migrations to a specific version.
func DownTo(db *sql.DB, dir string, version int64) error {
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	return migrations.DownTo(db, version)
}
