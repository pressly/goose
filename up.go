package goose

import (
	"database/sql"
	"fmt"
	"log"
)

// UpTo migrates up to a specific version.
func UpTo(db *sql.DB, dir string, version int64) error {
	migrations, err := CollectMigrations(dir, minVersion, version)
	if err != nil {
		return err
	}

	for {
		current, err := GetDBVersion(db)
		if err != nil {
			return err
		}

		next, err := migrations.Next(current)
		if err != nil {
			if err == ErrNoNextVersion {
				fmt.Printf("goose: no migrations to run. current version: %d\n", current)
				return nil
			}
			return err
		}

		if err = next.Up(db); err != nil {
			return err
		}
	}
}

// Up applies all available migrations.
func Up(db *sql.DB, dir string) error {
	return UpTo(db, dir, maxVersion)
}

// UpByOne migrates up by a single version.
func UpByOne(db *sql.DB, dir string) error {
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	currentVersion, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	next, err := migrations.Next(currentVersion)
	if err != nil {
		if err == ErrNoNextVersion {
			fmt.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
		}
		return err
	}

	if err = next.Up(db); err != nil {
		return err
	}

	return nil
}

// UpUnapplied migrates all un-migrated versions.
func UpUnapplied(db *sql.DB, dir string) error {

	// Collect all migrations from directory.
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		log.Print(err)
		return err
	}

	// Must ensure that the version table exists if we're running on a pristine DB.
	if _, err := EnsureDBVersion(db); err != nil {
		log.Print(err)
		return err
	}

	// Loop over each migration.
	for _, migration := range migrations {

		// Look up if the migration has been applied.
		var row MigrationRecord
		q := fmt.Sprintf("SELECT tstamp, is_applied FROM goose_db_version WHERE version_id=%d ORDER BY tstamp DESC LIMIT 1", migration.Version)
		err := db.QueryRow(q).Scan(&row.TStamp, &row.IsApplied)

		if err != nil && err != sql.ErrNoRows {
			log.Print(err)
			return err
		}

		// Only apply migrations which are not applied or found.
		if !row.IsApplied {
			if err = migration.Up(db); err != nil {
				log.Print(err)
				return err
			}
		}
	}
	return nil
}
