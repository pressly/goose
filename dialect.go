package goose

import (
	"fmt"

	"github.com/pressly/goose/v4/internal/dialectadapter"
)

// Dialect is the type of database dialect.
type Dialect string

const (
	DialectPostgres   Dialect = "postgres"
	DialectMySQL      Dialect = "mysql"
	DialectSQLite3    Dialect = "sqlite3"
	DialectMSSQL      Dialect = "mssql"
	DialectRedshift   Dialect = "redshift"
	DialectTiDB       Dialect = "tidb"
	DialectClickHouse Dialect = "clickhouse"
	DialectVertica    Dialect = "vertica"
)

var dialectLookup = map[Dialect]dialectadapter.Dialect{
	DialectPostgres:   dialectadapter.Postgres,
	DialectMySQL:      dialectadapter.Mysql,
	DialectSQLite3:    dialectadapter.Sqlite3,
	DialectMSSQL:      dialectadapter.Sqlserver,
	DialectRedshift:   dialectadapter.Redshift,
	DialectTiDB:       dialectadapter.Tidb,
	DialectClickHouse: dialectadapter.Clickhouse,
	DialectVertica:    dialectadapter.Vertica,
}

func ParseDialect(s string) (Dialect, error) {
	dialect := Dialect(s)
	if _, ok := dialectLookup[dialect]; !ok {
		return "", fmt.Errorf("unknown dialect: %s", s)
	}
	return dialect, nil
}
