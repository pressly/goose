package goose

import (
	"database/sql"
)

// UpTo migrates up to a specific version.
func UpTo(db *sql.DB, dir string, version int64) error { return def.UpTo(db, dir, version) }

// UpTo migrates up to a specific version.
func (in *Instance) UpTo(db *sql.DB, dir string, version int64) error {
	migrations, err := in.CollectMigrations(dir, minVersion, version)
	if err != nil {
		return err
	}

	for {
		current, err := in.GetDBVersion(db)
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

		if err = next.Up(db); err != nil {
			return err
		}
	}
}

// Up applies all available migrations.
func Up(db *sql.DB, dir string) error { return def.Up(db, dir) }

// Up applies all available migrations.
func (in *Instance) Up(db *sql.DB, dir string) error {
	return in.UpTo(db, dir, maxVersion)
}

// UpByOne migrates up by a single version.
func UpByOne(db *sql.DB, dir string) error { return def.UpByOne(db, dir) }

// UpByOne migrates up by a single version.
func (in *Instance) UpByOne(db *sql.DB, dir string) error {
	migrations, err := in.CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	currentVersion, err := in.GetDBVersion(db)
	if err != nil {
		return err
	}

	next, err := migrations.Next(currentVersion)
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
