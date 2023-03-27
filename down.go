package goose

import (
	"fmt"
	"github.com/pressly/goose/v3/internal"
)

// Down rolls back a single migration from the current version.
func Down(db internal.GooseDB, dir string, opts ...OptionsFunc) error {
	option := &options{}
	for _, f := range opts {
		f(option)
	}
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}
	if option.noVersioning {
		if len(migrations) == 0 {
			return nil
		}
		currentVersion := migrations[len(migrations)-1].Version
		// Migrate only the latest migration down.
		return downToNoVersioning(db, migrations, currentVersion-1)
	}
	currentVersion, err := GetDBVersion(db)
	if err != nil {
		return err
	}
	current, err := migrations.Current(currentVersion)
	if err != nil {
		return fmt.Errorf("no migration %v", currentVersion)
	}
	return current.Down(db)
}

// DownTo rolls back migrations to a specific version.
func DownTo(db internal.GooseDB, dir string, version int64, opts ...OptionsFunc) error {
	option := &options{}
	for _, f := range opts {
		f(option)
	}
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}
	if option.noVersioning {
		return downToNoVersioning(db, migrations, version)
	}

	for {
		currentVersion, err := GetDBVersion(db)
		if err != nil {
			return err
		}

		if currentVersion == 0 {
			log.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
			return nil
		}
		current, err := migrations.Current(currentVersion)
		if err != nil {
			log.Printf("goose: migration file not found for current version (%d), error: %s\n", currentVersion, err)
			return err
		}

		if current.Version <= version {
			log.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
			return nil
		}

		if err = current.Down(db); err != nil {
			return err
		}
	}
}

// downToNoVersioning applies down migrations down to, but not including, the
// target version.
func downToNoVersioning(db internal.GooseDB, migrations Migrations, version int64) error {
	var finalVersion int64
	for i := len(migrations) - 1; i >= 0; i-- {
		if version >= migrations[i].Version {
			finalVersion = migrations[i].Version
			break
		}
		migrations[i].noVersioning = true
		if err := migrations[i].Down(db); err != nil {
			return err
		}
	}
	log.Printf("goose: down to current file version: %d\n", finalVersion)
	return nil
}
