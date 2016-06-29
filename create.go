package goose

import (
	"database/sql"
	"fmt"
	"time"
)

// Create writes a new blank migration file.
func Create(db *sql.DB, name, migrationType, dir string) error {
	path, err := CreateMigration(name, migrationType, dir, time.Now())
	if err != nil {
		return err
	}
	fmt.Println(fmt.Sprintf("Created %s migration at %s", migrationType, path))

	return nil
}
