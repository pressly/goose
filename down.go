package goose

import (
	"database/sql"
	"fmt"
)

// Down rolls back a single migration from the current version.
func Down(db *sql.DB, dir string, opts ...OptionsFunc) error {
	return defaultProvider.Down(db, dir, opts...)
}

// Down rolls back a single migration from the current version.
func (p *Provider) Down(db *sql.DB, dir string, opts ...OptionsFunc) error {
	option := applyOptions(opts)
	migrations, err := p.CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}
	if option.noVersioning {
		if len(migrations) == 0 {
			return nil
		}
		currentVersion := migrations[len(migrations)-1].Version
		// Migrate only the latest migration down.
		return downToNoVersioning(p, db, migrations, currentVersion-1)
	}
	currentVersion, err := p.GetDBVersion(db)
	if err != nil {
		return err
	}
	current, err := migrations.Current(currentVersion)
	if err != nil {
		return fmt.Errorf("no migration %v", currentVersion)
	}
	return current.DownWithProvider(p, db)
}

// DownTo rolls back migrations to a specific version.
func DownTo(db *sql.DB, dir string, version int64, opts ...OptionsFunc) error {
	return defaultProvider.DownTo(db, dir, version, opts...)
}

// DownTo rolls back migrations to a specific version.
func (p *Provider) DownTo(db *sql.DB, dir string, version int64, opts ...OptionsFunc) error {
	option := applyOptions(opts)
	migrations, err := p.CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}
	if option.noVersioning {
		return downToNoVersioning(p, db, migrations, version)
	}

	for {
		currentVersion, err := p.GetDBVersion(db)
		if err != nil {
			return err
		}

		if currentVersion == 0 {
			p.log.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
			return nil
		}
		current, err := migrations.Current(currentVersion)
		if err != nil {
			p.log.Printf("goose: migration file not found for current version (%d), error: %s\n", currentVersion, err)
			return err
		}

		if current.Version <= version {
			p.log.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
			return nil
		}

		if err = current.DownWithProvider(p, db); err != nil {
			return err
		}
	}
}

// downToNoVersioning applies down migrations down to, but not including, the
// target version.
func downToNoVersioning(p *Provider, db *sql.DB, migrations Migrations, version int64) error {
	if p == nil {
		p = defaultProvider
	}
	var finalVersion int64
	for i := len(migrations) - 1; i >= 0; i-- {
		if version >= migrations[i].Version {
			finalVersion = migrations[i].Version
			break
		}
		migrations[i].noVersioning = true
		if err := migrations[i].DownWithProvider(p, db); err != nil {
			return err
		}
	}
	p.log.Printf("goose: down to current file version: %d\n", finalVersion)
	return nil
}
