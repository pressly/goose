package goose

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

// MigrationRecordWithSource struct.
type MigrationRecordWithSource struct {
	MigrationRecord
	Source string
}

var statusPrintFunc = func(migrations []*MigrationRecordWithSource) error {
	log.Println("    Applied At                  Migration")
	log.Println("    =======================================")
	for _, migration := range migrations {
		var appliedAt string
		if migration.IsApplied {
			appliedAt = migration.TStamp.Format(time.ANSIC)
		} else {
			appliedAt = "Pending"
		}

		log.Printf("    %-24s -- %v\n", appliedAt, filepath.Base(migration.Source))
	}

	return nil
}

// SetStatusPrintFunc set the status print function.
func SetStatusPrintFunc(f func([]*MigrationRecordWithSource) error) {
	statusPrintFunc = f
}

// Status prints the status of all migrations.
func Status(db *sql.DB, dir string) error {
	// collect all migrations
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return errors.Wrap(err, "failed to collect migrations")
	}

	// must ensure that the version table exists if we're running on a pristine DB
	if _, err := EnsureDBVersion(db); err != nil {
		return errors.Wrap(err, "failed to ensure DB version")
	}

	var ms []*MigrationRecordWithSource
	for _, migration := range migrations {
		r, err := getMigrationRecord(db, migration.Version)
		if err != nil {
			return errors.Wrapf(err, "failed to query migration, version: %d", migration.Version)
		}

		ms = append(ms, &MigrationRecordWithSource{MigrationRecord: *r, Source: migration.Source})
	}

	if err := statusPrintFunc(ms); err != nil {
		return errors.Wrap(err, "failed to print status")
	}

	return nil
}

func getMigrationRecord(db *sql.DB, version int64) (*MigrationRecord, error) {
	q := fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id=%d ORDER BY tstamp DESC LIMIT 1", TableName(), version)

	var row MigrationRecord
	err := db.QueryRow(q).Scan(&row.TStamp, &row.IsApplied)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	row.VersionID = version
	return &row, nil
}
