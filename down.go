package goose

import (
	"fmt"
)

// Down rolls back a single migration from the current version.
func Down(opts *options) error {
	currentVersion, err := GetDBVersion(opts.db)
	if err != nil {
		return err
	}

	migrations, err := CollectMigrations(opts, minVersion, maxVersion)
	if err != nil {
		return err
	}

	current, err := migrations.Current(currentVersion)
	if err != nil {
		return fmt.Errorf("no migration %v", currentVersion)
	}

	return current.Down(opts.db)
}

// DownTo rolls back migrations to a specific version.
func DownTo(opts *options, version int64) error {
	migrations, err := CollectMigrations(opts, minVersion, maxVersion)
	if err != nil {
		return err
	}

	for {
		currentVersion, err := GetDBVersion(opts.db)
		if err != nil {
			return err
		}

		current, err := migrations.Current(currentVersion)
		if err != nil {
			log.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
			return nil
		}

		if current.Version <= version {
			log.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
			return nil
		}

		if err = current.Down(opts.db); err != nil {
			return err
		}
	}
}
