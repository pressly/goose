package goose

import (
	"fmt"
)

// Down rolls back a single migration from the current version.
func Down(cfg *config) error {
	currentVersion, err := GetDBVersion(cfg.db)
	if err != nil {
		return err
	}

	migrations, err := CollectMigrations(cfg, minVersion, maxVersion)
	if err != nil {
		return err
	}

	current, err := migrations.Current(currentVersion)
	if err != nil {
		return fmt.Errorf("no migration %v", currentVersion)
	}

	return current.Down(cfg)
}

// DownTo rolls back migrations to a specific version.
func DownTo(cfg *config, version int64) error {
	migrations, err := CollectMigrations(cfg, minVersion, maxVersion)
	if err != nil {
		return err
	}

	for {
		currentVersion, err := GetDBVersion(cfg.db)
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

		if err = current.Down(cfg); err != nil {
			return err
		}
	}
}
