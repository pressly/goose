package goose

import (
	"database/sql"
	"fmt"
	"time"
)

// Create writes a new blank migration file.
func Create(db *sql.DB, dir, name, migrationType string, sequential bool) error {
	var version string
	if !sequential {
		version = time.Now().Format("20060102150405")
	} else {
		m, err := LastMigration(dir)
		var last int64
		if err != nil {
			if err != ErrNoNextVersion {
				return err
			}
			last = 0;
		} else {
			last = m.Version
		}
		version = fmt.Sprintf("%d", last + 1)
	}
	path, err := CreateMigration(name, migrationType, dir, version)
	if err != nil {
		return err
	}
	fmt.Println(fmt.Sprintf("Created %s migration at %s", migrationType, path))

	return nil
}
