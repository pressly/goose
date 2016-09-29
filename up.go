package goose

import (
	"database/sql"
	"fmt"
)

func Up(db *sql.DB, dir string) error {
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}
	migrations.Sort(true)

	target, err := migrations.Last()
	if err != nil {
		return err
	}

	if err := RunMigrations(db, dir, target); err != nil {
		return err
	}
	return nil
}

func UpByOne(db *sql.DB, dir string) error {
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}
	migrations.Sort(true)

	current, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	next, err := migrations.Next(current)
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
