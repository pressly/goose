package goose

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
)

func Fix(db *sql.DB, dir string) error {
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	// split into timestamped and versioned migrations
	tsMigrations, err := migrations.Timestamped()
	if err != nil {
		return err
	}

	vMigrations, err := migrations.Versioned()
	if err != nil {
		return err
	}
	// Initial version.
	version := int64(1)
	if last, err := vMigrations.Last(); err == nil {
		version = last.Version + 1
	}

	// fix filenames by replacing timestamps with sequential versions
	for _, tsm := range tsMigrations {
		oldPath := tsm.Source
		newPath := strings.Replace(oldPath, fmt.Sprintf("%d", tsm.Version), fmt.Sprintf("%05v", version), 1)

		if err := os.Rename(oldPath, newPath); err != nil {
			return err
		}

		version++
	}

	return nil
}
