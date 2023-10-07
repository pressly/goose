package goose

import (
	"fmt"

	"github.com/pressly/goose/v3/internal/dialect"
)

// Dialect is the type of database dialect.
type Dialect string

const (
	DialectClickHouse Dialect = "clickhouse"
	DialectMSSQL      Dialect = "mssql"
	DialectMySQL      Dialect = "mysql"
	DialectPostgres   Dialect = "postgres"
	DialectRedshift   Dialect = "redshift"
	DialectSQLite3    Dialect = "sqlite3"
	DialectTiDB       Dialect = "tidb"
	DialectVertica    Dialect = "vertica"
)

func init() {
	store, _ = dialect.NewStore(dialect.Postgres)
}

var store dialect.Store

// SetDialect sets the dialect to use for the goose package.
func SetDialect(s string) error {
	var d dialect.Dialect
	switch s {
	case "postgres", "pgx":
		d = dialect.Postgres
	case "mysql":
		d = dialect.Mysql
	case "sqlite3", "sqlite":
		d = dialect.Sqlite3
	case "mssql", "azuresql", "sqlserver":
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
	store, err = dialect.NewStore(d)
	return err
}
