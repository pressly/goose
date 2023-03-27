package goose

import (
	"database/sql"
	"fmt"
)

// OpenDBWithDriver creates a connection to a database, and modifies goose
// internals to be compatible with the supplied driver by calling SetDialect.
func OpenDBWithDriver(driver string, dbstring string) (Connection, error) {
	if err := SetDialect(driver); err != nil {
		return nil, err
	}

	switch driver {
	case "mssql":
		driver = "sqlserver"
	case "redshift":
		driver = "postgres"
	case "tidb":
		driver = "mysql"
	}

	switch driver {
	case "postgres", "pgx", "sqlite3", "sqlite", "mysql", "sqlserver", "clickhouse", "vertica":
		conn, err := sql.Open(driver, dbstring)
		return SqlDbToGooseAdapter{Conn: conn}, err
	default:
		return nil, fmt.Errorf("unsupported driver %s", driver)
	}
}
