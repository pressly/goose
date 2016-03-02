package goose

import (
	"database/sql"
)

func Up(db *sql.DB, dir string) error {
	target, err := GetMostRecentDBVersion(dir)
	if err != nil {
		return err
	}

	if err := RunMigrations(db, dir, target); err != nil {
		return err
	}
	return nil
}
