package goose

import (
	"database/sql"
	"fmt"
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

// UpMissing migrates all missing migrations
func UpMissing(db *sql.DB, dir string) error {
	migrations, err := MissingMigrations(db, dir)
	if err != nil {
		return err
	}

	// must ensure that the version table exists if we're running on a pristine DB
	if _, err := EnsureDBVersion(db); err != nil {
		return err
	}

	if len(migrations) == 0 {
		fmt.Printf("goose: no missing migrations to run\n")
	}

	for _, migration := range migrations {
		if err = migration.Up(db); err != nil {
			return err
		}
	}

	return nil
}

// UpWithMissing migrates all missing migrations, then all new migrations
func UpWithMissing(db *sql.DB, dir string) error {
	err := UpMissing(db, dir)
	if err != nil {
		return err
	}

	return Up(db, dir)
}
