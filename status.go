package goose

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"
)

// Status prints the status of all migrations.
func Status(db *sql.DB, dir string, opts ...OptionsFunc) error {
	ctx := context.Background()
	return StatusContext(ctx, db, dir, opts...)
}

// StatusContext prints the status of all migrations.
func StatusContext(ctx context.Context, db *sql.DB, dir string, opts ...OptionsFunc) error {
	option := &options{}
	for _, f := range opts {
		f(option)
	}
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return fmt.Errorf("failed to collect migrations: %w", err)
	}
	if option.noVersioning {
		log.Printf("    Applied At                  Migration")
		log.Printf("    =======================================")
		for _, current := range migrations {
			log.Printf("    %-24s -- %v", "no versioning", filepath.Base(current.Source))
		}
		return nil
	}

	// must ensure that the version table exists if we're running on a pristine DB
	if _, err := EnsureDBVersionContext(ctx, db); err != nil {
		return fmt.Errorf("failed to ensure DB version: %w", err)
	}

	// Fetch all migrations from the database in a single query
	dbMigrations, err := store.ListMigrations(ctx, db, TableName())
	if err != nil {
		return fmt.Errorf("failed to list migrations: %w", err)
	}

	// Build a map of version_id to migration result for quick lookup
	dbMigrationMap := make(map[int64]*struct {
		Timestamp time.Time
		IsApplied bool
	})
	for _, m := range dbMigrations {
		dbMigrationMap[m.VersionID] = &struct {
			Timestamp time.Time
			IsApplied bool
		}{
			Timestamp: m.Timestamp,
			IsApplied: m.IsApplied,
		}
	}

	log.Printf("    Applied At                  Migration")
	log.Printf("    =======================================")
	for _, migration := range migrations {
		appliedAt := "Pending"
		if m, exists := dbMigrationMap[migration.Version]; exists && m.IsApplied {
			appliedAt = m.Timestamp.Format(time.ANSIC)
		}
		log.Printf("    %-24s -- %v", appliedAt, filepath.Base(migration.Source))
	}

	return nil
}
