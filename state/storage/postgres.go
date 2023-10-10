package storage

import (
	"fmt"

	"github.com/pressly/goose/v3/state"
)

// PostgreSQL is a storage implementation for PostgreSQL.
// Pass an empty table name to use the default "goose_db_version" table name.
//
// Experimental: This API is experimental and may change in the future.
func PostgreSQL(tableName string) state.Storage {
	tableName = defaultTablename(tableName)
	return queries{
		createTable: fmt.Sprintf(`CREATE TABLE %s (
			id serial NOT NULL,
			version_id bigint NOT NULL,
			is_applied boolean NOT NULL,
			tstamp timestamp NULL default now(),
			PRIMARY KEY(id)
		)`, tableName),
		insertVersion:         fmt.Sprintf(`INSERT INTO %s (version_id, is_applied) VALUES ($1, $2)`, tableName),
		deleteVersion:         fmt.Sprintf(`DELETE FROM %s WHERE version_id=$1`, tableName),
		getMigrationByVersion: fmt.Sprintf(`SELECT tstamp, is_applied FROM %s WHERE version_id=$1 ORDER BY tstamp DESC LIMIT 1`, tableName),
		listMigrations:        fmt.Sprintf(`SELECT version_id, is_applied from %s ORDER BY id DESC`, tableName),
	}
}
