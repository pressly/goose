package goose

import (
	"database/sql"
	"fmt"
	"sort"
)

// Down rolls back a single migration from the current version.
func Down(db *sql.DB, dir string, opts ...OptionsFunc) error {
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
		return downNoVersioning(db, migrations, migrations[len(migrations)-1].Version)
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
func DownTo(db *sql.DB, dir string, version int64, opts ...OptionsFunc) error {
	option := &options{}
	for _, f := range opts {
		f(option)
	}
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}
	if option.noVersioning {
		return downNoVersioning(db, migrations, version)
	}

	for {
		currentVersion, err := GetDBVersion(db)
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

		if err = current.Down(db); err != nil {
			return err
		}
	}
}

func downNoVersioning(db *sql.DB, migrations Migrations, version int64) error {
	// TODO(mf): we're not tracking the seed migrations in the database,
	// which means subsequent "down" operations will always start from the
	// highest seed file.
	// Also, should target version always be 0 and error otherwise?
	sort.Sort(sort.Reverse(migrations))

	var finalVersion int64
	for _, current := range migrations {
		if current.Version <= version {
			log.Printf("goose: current version: %d\n", current.Version)
			return nil
		}
		current.noVersioning = true
		if err := current.Down(db); err != nil {
			return err
		}
		finalVersion = current.Version
	}
	finalVersion--
	log.Printf("goose: current version: %d\n", finalVersion)
	return nil
}
