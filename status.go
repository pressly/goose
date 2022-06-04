package goose

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"
)

// Status prints the status of all migrations.
func Status(db *sql.DB, dir string, opts ...OptionsFunc) error {
	return defaultProvider.Status(db, dir, opts...)
}

func (p *Provider) Status(db *sql.DB, dir string, opts ...OptionsFunc) error {
	option := applyOptions(opts)
	migrations, err := p.CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return fmt.Errorf("failed to collect migrations: %w", err)
	}
	if option.noVersioning {
		p.log.Println("    Applied At                  Migration")
		p.log.Println("    =======================================")
		for _, current := range migrations {
			p.log.Printf("    %-24s -- %v\n", "no versioning", filepath.Base(current.Source))
		}
		return nil
	}

	// must ensure that the version table exists if we're running on a pristine DB
	if _, err := p.EnsureDBVersion(db); err != nil {
		return fmt.Errorf("failed to ensure DB version: %w", err)
	}

	p.log.Println("    Applied At                  Migration")
	p.log.Println("    =======================================")
	for _, migration := range migrations {
		if err := p.printMigrationStatus(db, migration.Version, filepath.Base(migration.Source)); err != nil {
			return fmt.Errorf("failed to print status: %w", err)
		}
	}

	return nil
}

func (p *Provider) printMigrationStatus(db *sql.DB, version int64, script string) error {
	q := p.dialect.migrationSQL()

	var row MigrationRecord

	err := db.QueryRow(q, version).Scan(&row.TStamp, &row.IsApplied)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to query the latest migration: %w", err)
	}

	var appliedAt string
	if row.IsApplied {
		appliedAt = row.TStamp.Format(time.ANSIC)
	} else {
		appliedAt = "Pending"
	}

	p.log.Printf("    %-24s -- %v\n", appliedAt, script)
	return nil
}
