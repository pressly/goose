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

var dbVersionTableName = "goose_db_version"

// GetDBVersionTableName returns goose db version table name
func GetDBVersionTableName() string {
	return dbVersionTableName
}

// SetDBVersionTableName set goose db version table name
func SetDBVersionTableName(n string) {
	dbVersionTableName = n
}
