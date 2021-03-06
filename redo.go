package goose

import (
	"database/sql"
)

// Redo rolls back the most recently applied migration, then runs it again.
func (ms Migrations) Redo(db *sql.DB) error {
	currentVersion, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	current, err := ms.Current(currentVersion)
	if err != nil {
		return err
	}

	if err := current.Down(db); err != nil {
		return err
	}

	if err := current.Up(db); err != nil {
		return err
	}

	return nil
}

// Redo rolls back the most recently applied migration, then runs it again.
func Redo(db *sql.DB, dir string) error {
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	return migrations.Redo(db)
}
