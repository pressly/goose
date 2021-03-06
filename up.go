package goose

import (
	"database/sql"
)

// UpByOne migrates up by a single version.
func (ms Migrations) UpByOne(db *sql.DB) error {
	currentVersion, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	next, err := ms.Next(currentVersion)
	if err != nil {
		if err == ErrNoNextVersion {
			log.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
		}
		return err
	}

	if err = next.Up(db); err != nil {
		return err
	}

	return nil
}

// Up applies all available migrations.
func (ms Migrations) Up(db *sql.DB) error {
	for {
		err := ms.UpByOne(db)
		switch {
		case err == nil:
		case err == ErrNoNextVersion:
			return nil
		default:
			return err
		}
	}
}

// UpTo migrates up to a specific version.
func UpTo(db *sql.DB, dir string, version int64) error {
	migrations, err := CollectMigrations(dir, minVersion, version)
	if err != nil {
		return err
	}

	return migrations.Up(db)
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

	return migrations.UpByOne(db)
}
