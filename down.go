package goose

import (
	"database/sql"
	"fmt"
)

func Down(db *sql.DB, dir string) error {
	current, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	previous, err := GetPreviousDBVersion(dir, current)
	if err != nil {
		if err != nil {
			if err == ErrNoPreviousVersion {
				fmt.Printf("goose: no migrations to run. current version: %d\n", current)
			}
			return err
		}

		return err
	}

	if err = RunMigrations(db, dir, previous); err != nil {
		return err
	}

	return nil
}
