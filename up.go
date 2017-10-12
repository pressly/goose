package goose

import (
	"database/sql"
	"fmt"
)

// Up applies all available migrations.
func Up(db *sql.DB, dir string) error {
	return UpMissing(db, dir, false)
}

// UpByOne migrates up by a single version.
func UpByOne(db *sql.DB, dir string) error {
	return UpMissing(db, dir, true)
}

// UpMissing migrates all missing migrations
func UpMissing(db *sql.DB, dir string, onlyOne bool) error {
	migrations, err := MissingMigrations(db, dir)
	if err != nil {
		return err
	}

	if len(migrations) == 0 {
		fmt.Printf("goose: no migrations to run\n")
	}

	for _, migration := range migrations {
		if err = migration.Up(db); err != nil {
			return err
		}
		if onlyOne {
			break
		}
	}

	return nil
}

// UpTo migrates up to a specific version.
func UpTo(db *sql.DB, dir string, version int64) error {
	migrations, err := MissingMigrations(db, dir)
	if err != nil {
		return err
	}

	if len(migrations) == 0 {
		fmt.Printf("goose: no migrations to run\n")
	}

	for _, migration := range migrations {
		if migration.Version > version {
			break
		}
		if err = migration.Up(db); err != nil {
			return err
		}
	}

	return nil
}
