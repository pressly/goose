package goose

import (
	"database/sql"
	"fmt"
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

	next, err := GetNextDBVersion(dir, current)
	if err != nil {
		if err == ErrNoNextVersion {
			fmt.Printf("goose: no migrations to run. current version: %d\n", current)
		}
		return err
	}

	if err = RunMigrations(db, dir, next); err != nil {
		return err
	}

	return nil
}
