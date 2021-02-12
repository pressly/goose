package goose

import (
	"database/sql"
	"fmt"
)

// OpenDBWithDriver creates a connection a database, and modifies goose
// internals to be compatible with the supplied driver by calling SetDialect.
func OpenDBWithDriver(driver string, dbstring string) (*sql.DB, error) {
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
	case "clickhouse", "mysql", "postgres", "snowflake", "sqlite3", "sqlserver":
		return sql.Open(driver, dbstring)
	default:
		return nil, fmt.Errorf("unsupported driver %s", driver)
	}
}
