package goose

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/pressly/goose/v3/internal/legacystore"
)

type statusLine struct {
	Version   int64
	AppliedAt time.Time
	Pending   bool
	Source    string
}

type statusLines []*statusLine

// helpers so we can use pkg sort
func (s statusLines) Len() int      { return len(s) }
func (s statusLines) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s statusLines) Less(i, j int) bool {
	lineI := s[i]
	lineJ := s[j]
	// Pending migrations always come later:
	if lineI.Pending != lineJ.Pending {
		// lineJ is pending ---> lineI goes first and return value true means lineI < lineJ
		return lineJ.Pending
	}
	if !lineI.AppliedAt.Equal(lineJ.AppliedAt) {
		return lineI.AppliedAt.Before(lineJ.AppliedAt)
	}
	if lineI.Version != lineJ.Version {
		return lineI.Version < lineJ.Version
	}
	return lineI.Source < lineJ.Source
}

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
	fsMigrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return fmt.Errorf("failed to collect migrations: %w", err)
	}
	if option.noVersioning {
		log.Printf("    Applied At                  Migration")
		log.Printf("    =======================================")
		for _, current := range fsMigrations {
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

	// Build a map on version_id to match migrations in DB to migrations from FS.
	dbMigrationMap := make(map[int64]*legacystore.ListMigrationsResult)
	for _, m := range dbMigrations {
		dbMigrationMap[m.VersionID] = m
	}

	// Gather 1 status line for each migration in the FS, enriched with application timestamp from DB if applied:
	var statusOutput statusLines
	for _, fsM := range fsMigrations {
		line := statusLine{
			Version:   fsM.Version,
			AppliedAt: time.Time{},
			Pending:   true,
			Source:    filepath.Base(fsM.Source),
		}
		if dbM, exists := dbMigrationMap[fsM.Version]; exists && dbM.IsApplied {
			line.Pending = false
			line.AppliedAt = dbM.Timestamp
		}
		statusOutput = append(statusOutput, &line)
	}
	sort.Sort(statusOutput)

	log.Printf("    Applied At                  Migration")
	log.Printf("    =======================================")
	for _, migration := range statusOutput {
		appliedAt := "Pending"
		if !migration.Pending {
			appliedAt = migration.AppliedAt.Format(time.ANSIC)
		}
		log.Printf("    %-24s -- %v", appliedAt, migration.Source)
	}

	return nil
}
