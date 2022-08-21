package goose

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"
)

// Status prints the status of all migrations.
func Status(db *sql.DB, dir string, opts ...OptionsFunc) error {
	option := &options{}
	for _, f := range opts {
		f(option)
	}
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return fmt.Errorf("failed to collect migrations: %w", err)
	}
	if option.noVersioning {
		log.Println("    Applied At                  Migration")
		log.Println("    =======================================")
		for _, current := range migrations {
			log.Printf("    %-24s -- %v\n", "no versioning", filepath.Base(current.Source))
		}
		return nil
	}

	// must ensure that the version table exists if we're running on a pristine DB
	if _, err := EnsureDBVersion(db); err != nil {
		return fmt.Errorf("failed to ensure DB version: %w", err)
	}

	log.Println("    Applied At                  Migration")
	log.Println("    =======================================")
	for _, migration := range migrations {
		if err := printMigrationStatus(db, migration.Version, filepath.Base(migration.Source)); err != nil {
			return fmt.Errorf("failed to print status: %w", err)
		}
	}

	return nil
}

func printMigrationStatus(db *sql.DB, version int64, script string) error {
	row, err := GetMigrationRecord(db, version)
	if err != nil {
		return err
	}

	appliedAt := "Pending"
	if row != nil && row.IsApplied {
		appliedAt = row.TStamp.Format(time.ANSIC)
	}

	log.Printf("    %-24s -- %v\n", appliedAt, script)
	return nil
}

func DetailedStatus(db *sql.DB, dir string, opts ...OptionsFunc) (Migrations, error) {
	option := &options{}
	for _, f := range opts {
		f(option)
	}
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to collect migrations: %w", err)
	}
	if option.noVersioning {
		return migrations, nil
	}

	// must ensure that the version table exists if we're running on a pristine DB
	if _, err := EnsureDBVersion(db); err != nil {
		return nil, fmt.Errorf("failed to ensure DB version: %w", err)
	}

	for i := range migrations {
		record, err := GetMigrationRecord(db, migrations[i].Version)
		if err != nil {
			return nil, err
		}

		if record == nil {
			migrations[i].IsApplied = false
			continue
		}

		migrations[i].IsApplied = record.IsApplied
		migrations[i].TStamp = record.TStamp
	}

	return migrations, nil
}

// GetMigrationRecord - returns the migration record for a given version.
// If no record is found, returns nil.
func GetMigrationRecord(db *sql.DB, version int64) (*MigrationRecord, error) {
	q := GetDialect().migrationSQL()

	row := MigrationRecord{VersionID: version}

	err := db.QueryRow(q, version).Scan(&row.TStamp, &row.IsApplied)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("failed to query the latest migration: %w", err)
	}

	return &row, nil
}
