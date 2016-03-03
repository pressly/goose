package goose

import (
	"database/sql"
	"fmt"
)

func Version(db *sql.DB, dir string) error {
	current, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	fmt.Printf("goose: dbversion %v\n", current)
	return nil
}
