package goose

import (
	"database/sql"
)

func Redo(db *sql.DB, dir string) error {
	currentVersion, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	migrations, err := collectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	current, err := migrations.Current(currentVersion)
	if err != nil {
		return err
	}

	previous, err := migrations.Next(currentVersion)
	if err != nil {
		return err
	}

	if err := previous.Up(db); err != nil {
		return err
	}

	if err := current.Up(db); err != nil {
		return err
	}

	return nil
}
