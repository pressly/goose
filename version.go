package goose

import (
	"database/sql"
	"fmt"
)

// Version prints the current version of the database.
func Version(db *sql.DB, dir string) error {
	current, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	fmt.Printf("goose: version %v\n", current)
	return nil
}
