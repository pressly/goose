package goose

import (
	"database/sql"
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
			log.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
		}
		return err
	}

	if err = next.Up(db); err != nil {
		return err
	}

	return nil
}

// UpAll migrates up all pending migrations.
func UpAll(db *sql.DB, dir string) error {
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}
	versions, err := EnsureDBVersions(db)
	if err != nil {
		return err
	}
	for _, migration := range migrations {
		if versions[migration.Version] {
			continue
		}
		if err = migration.Up(db); err != nil {
			return err
		}
	}
	return nil
}
