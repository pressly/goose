package goose

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"
)

// Status prints the status of all migrations.
func Status(db *sql.DB, dir string, opts ...OptionsFunc) error {
	ctx := context.Background()
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
		if err := printMigrationStatus(ctx, db, migration.Version, filepath.Base(migration.Source)); err != nil {
			return fmt.Errorf("failed to print status: %w", err)
		}
	}

	return nil
}

func printMigrationStatus(ctx context.Context, db *sql.DB, version int64, script string) error {
	m, err := store.GetMigration(ctx, db, version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to query the latest migration: %w", err)
	}
	appliedAt := "Pending"
	if m != nil && m.IsApplied {
		appliedAt = m.Timestamp.Format(time.ANSIC)
	}
	log.Printf("    %-24s -- %v\n", appliedAt, script)
	return nil
}
