package storage

import (
	"fmt"

	"github.com/pressly/goose/v3/state"
)

// Sqlite3 is a storage implementation for sqlite
// which uses the "goose_db_version" table name to store the migration state.
//
// Experimental: This API is experimental and may change in the future.
func Sqlite3() state.Storage {
	return Sqlite3WithTableName(defaultTablename)
}

// Sqlite3 is a storage implementation for sqlite.
// Sepicify the name of the table to store the migration state.
//
// Experimental: This API is experimental and may change in the future.
func Sqlite3WithTableName(tableName string) state.Storage {
	return queries{
		createTable: fmt.Sprintf(`CREATE TABLE %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			version_id INTEGER NOT NULL,
			is_applied INTEGER NOT NULL,
			tstamp TIMESTAMP DEFAULT (datetime('now'))
		)`, tableName),
		insertVersion:         fmt.Sprintf(`INSERT INTO %s (version_id, is_applied) VALUES (?, ?)`, tableName),
		deleteVersion:         fmt.Sprintf(`DELETE FROM %s WHERE version_id=?`, tableName),
		getMigrationByVersion: fmt.Sprintf(`SELECT tstamp, is_applied FROM %s WHERE version_id=? ORDER BY tstamp DESC LIMIT 1`, tableName),
		listMigrations:        fmt.Sprintf(`SELECT version_id, is_applied from %s ORDER BY id DESC`, tableName),
	}
}
