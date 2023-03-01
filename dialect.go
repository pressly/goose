package goose

import (
	"fmt"

	"github.com/pressly/goose/v3/internal/dialect"
)

func init() {
	sqlDialect, _ = dialect.NewSQLDialect(dialect.Postgres, TableName())
}

var sqlDialect dialect.SQLDialect

func getDialect() dialect.SQLDialect {
	return sqlDialect
}

// SetDialect sets the SQLDialect to use for the goose package.
func SetDialect(s string) error {
	var d dialect.Dialect
	switch s {
	case "postgres", "pgx":
		d = dialect.Postgres
	case "mysql":
		d = dialect.Mysql
	case "sqlite3", "sqlite":
		d = dialect.Sqlite3
	case "mssql":
		d = dialect.Sqlserver
	case "redshift":
		d = dialect.Redshift
	case "tidb":
		d = dialect.Tidb
	case "clickhouse":
		d = dialect.Clickhouse
	case "vertica":
		d = dialect.Vertica
	default:
		return fmt.Errorf("%q: unknown dialect", s)
	}
	var err error
	sqlDialect, err = dialect.NewSQLDialect(d, TableName())
	return err
}
