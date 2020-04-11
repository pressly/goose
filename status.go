package goose

import (
	"database/sql"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

// Status prints the status of all migrations.
func Status(db *sql.DB, dir string) error { return def.Status(db, dir) }

// Status prints the status of all migrations.
func (in *Instance) Status(db *sql.DB, dir string) error {
	// collect all migrations
	migrations, err := in.CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return errors.Wrap(err, "failed to collect migrations")
	}

	// must ensure that the version table exists if we're running on a pristine DB
	if _, err := in.EnsureDBVersion(db); err != nil {
		return errors.Wrap(err, "failed to ensure DB version")
	}

	in.log.Println("    Applied At                  Migration")
	in.log.Println("    =======================================")
	for _, migration := range migrations {
		if err := in.printMigrationStatus(db, migration.Version, filepath.Base(migration.Source)); err != nil {
			return errors.Wrap(err, "failed to print status")
		}
	}

	return nil
}

func (in *Instance) printMigrationStatus(db *sql.DB, version int64, script string) error {
	q := in.GetDialect().migrationSQL()

	var row MigrationRecord

	err := db.QueryRow(q, version).Scan(&row.TStamp, &row.IsApplied)
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "failed to query the latest migration")
	}

	var appliedAt string
	if row.IsApplied {
		appliedAt = row.TStamp.Format(time.ANSIC)
	} else {
		appliedAt = "Pending"
	}

	in.log.Printf("    %-24s -- %v\n", appliedAt, script)
	return nil
}
