package goose

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/pressly/goose/v3/internal"
)

// OpenDBWithDriver creates a connection to a database, and modifies goose
// internals to be compatible with the supplied driver by calling SetDialect.
func OpenDBWithDriver(driver string, dbstring string) (internal.GooseDB, error) {
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
	case "postgres", "pgx": //,
		conn, err := pgx.Connect(context.Background(), dbstring)
		return internal.PgxToGooseAdapter{Conn: conn}, err
	case "sqlite3", "sqlite", "mysql", "sqlserver", "clickhouse", "vertica":
		conn, err := sql.Open(driver, dbstring)
		return internal.SqlToGooseAdapter{Conn: conn}, err
	default:
		return nil, fmt.Errorf("unsupported driver %s", driver)
	}
}
