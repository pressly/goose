package goose

import (
	"database/sql"
)

// Version prints the current version of the database.
func Version(db *sql.DB, dir string) error { return def.Version(db, dir) }

// Version prints the current version of the database.
func (in *Instance) Version(db *sql.DB, dir string) error {
	current, err := in.GetDBVersion(db)
	if err != nil {
		return err
	}

	log.Printf("goose: version %v\n", current)
	return nil
}

// TableName returns goose db version table name
func TableName() string { return def.TableName() }

// TableName returns goose db version table name
func (in *Instance) TableName() string {
	return in.tableName
}

// SetTableName set goose db version table name
func SetTableName(n string) { def.SetTableName(n) }

// SetTableName set goose db version table name
func (in *Instance) SetTableName(n string) {
	in.tableName = n
}
