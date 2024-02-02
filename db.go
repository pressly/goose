package goose

import (
	"database/sql"
	"fmt"
)

// OpenDBWithDriver creates a connection to a database, and modifies goose
// internals to be compatible with the supplied driver by calling SetDialect.
func OpenDBWithDriver(driver string, dbstring string) (*sql.DB, error) {
	if err := SetDialect(driver); err != nil {
		return nil, err
	}

	// To avoid breaking existing consumers. An implementation detail
	// that consumers should not care which underlying driver is used.
	switch driver {
	case "mssql":
		driver = "sqlserver"
	case "tidb":
		driver = "mysql"
	case "turso":
		driver = "libsql"
	case "sqlite3":
		//  Internally uses the CGo-free port of SQLite: modernc.org/sqlite
		driver = "sqlite"
	case "postgres", "redshift":
		driver = "pgx"
	}

	switch driver {
	case "postgres", "pgx", "sqlite3", "sqlite", "mysql", "sqlserver", "clickhouse", "vertica", "azuresql", "ydb", "libsql":
		return sql.Open(driver, dbstring)
	default:
		return nil, fmt.Errorf("unsupported driver %s", driver)
	}
}
