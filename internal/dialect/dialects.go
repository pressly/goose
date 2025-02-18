package dialect

import "errors"

// Dialect is the type of database dialect.
type Dialect string

var ErrUnknownDialect = errors.New("unknown dialect")

const (
	Postgres Dialect = "postgres"
	Mysql    Dialect = "mysql"
	Sqlite3  Dialect = "sqlite3"
	Mssql    Dialect = "mssql"
	// Deprecated: use [Mssql]
	Sqlserver  Dialect = "sqlserver"
	Redshift   Dialect = "redshift"
	Tidb       Dialect = "tidb"
	Clickhouse Dialect = "clickhouse"
	Vertica    Dialect = "vertica"
	Ydb        Dialect = "ydb"
	Turso      Dialect = "turso"
	Starrocks  Dialect = "starrocks"
)
