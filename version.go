package goose

import (
	"database/sql"
	"log"
)

// Version prints the current version of the database.
func Version(db *sql.DB, dir string) error {
	current, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	log.Printf("goose: version %v\n", current)
	return nil
}
