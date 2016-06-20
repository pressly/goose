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

func UpByOne(db *sql.DB, dir string) error {
	current, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	next, _ := GetNextDBVersion(dir, current)
	if err = RunMigrations(db, dir, next); err != nil {
		return err
	}

	return nil
}
