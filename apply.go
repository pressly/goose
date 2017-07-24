package goose

import (
	"database/sql"
	"fmt"
)

// Apply migrates a specific version migration.
func Apply(db *sql.DB, dir string, version int64) error {
	migrations, err := CollectMigrations(dir, version-1, version)
	if err != nil {
		return err
	}
	if l := len(migrations); l != 1 {
		return fmt.Errorf("expected one migration - found %v", l)
	}

	// Ensure goose_db_version table is present.
	_, err = GetDBVersion(db)
	if err != nil {
		return err
	}

	// Apply returned migration.
	for _, m := range migrations {
		if err = m.Up(db); err != nil {
			return err
		}
	}
	return nil
}

// Apply migrates a specific version migration.
func Revert(db *sql.DB, dir string, version int64) error {
	migrations, err := CollectMigrations(dir, version-1, version)
	if err != nil {
		return err
	}
	if l := len(migrations); l != 1 {
		return fmt.Errorf("expected one migration - found %v", l)
	}

	// Ensure goose_db_version table is present.
	_, err = GetDBVersion(db)
	if err != nil {
		return err
	}

	// Apply returned migration.
	for _, m := range migrations {
		if err = m.Down(db); err != nil {
			return err
		}
	}
	return nil
}
