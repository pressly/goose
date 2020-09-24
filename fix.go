package goose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Fix will rename migration files to fix the order
func Fix(cfg *config) error {
	migrations, err := CollectMigrations(cfg, minVersion, maxVersion)
	if err != nil {
		return err
	}

	// split into timestamped and versioned migrations
	tsMigrations, err := migrations.timestamped()
	if err != nil {
		return err
	}

	vMigrations, err := migrations.versioned()
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

		log.Printf("RENAMED %s => %s", filepath.Base(oldPath), filepath.Base(newPath))
		version++
	}

	return nil
}
