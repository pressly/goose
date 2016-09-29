package goose

import (
	"database/sql"
)

func Redo(db *sql.DB, dir string) error {
	current, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}
	migrations.Sort(false) // descending, Next will be Previous

	previous, err := migrations.Next(current)
	if err != nil {
		return err
	}

	if err := RunMigrations(db, dir, previous); err != nil {
		return err
	}

	if err := RunMigrations(db, dir, current); err != nil {
		return err
	}

	return nil
}
