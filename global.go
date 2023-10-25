package goose

import (
	"fmt"

	"github.com/pressly/goose/v3/state"
	"github.com/pressly/goose/v3/state/storage"
)

var global = struct {
	storageFactory func(string) state.Storage
	tableName      string
}{
	storageFactory: storage.PostgreSQLWithTableName,
	tableName:      "goose_db_version",
}

func globalStorage() state.Storage {
	return global.storageFactory(global.tableName)
}

// TableName returns goose db version table name
func TableName() string {
	return global.tableName
}

// SetTableName set goose db version table name
func SetTableName(n string) {
	global.tableName = n
}

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

// SetDialect sets the dialect to use for the goose package.
func SetDialect(s string) error {
	switch s {
	case "postgres", "pgx":
		global.storageFactory = storage.PostgreSQLWithTableName
	// case "mysql":
	// 	d = dialect.Mysql
	case "sqlite3", "sqlite":
		global.storageFactory = storage.Sqlite3WithTableName
	// case "mssql", "azuresql", "sqlserver":
	// d = dialect.Sqlserver
	// case "redshift":
	// d = dialect.Redshift
	// case "tidb":
	// d = dialect.Tidb
	// case "clickhouse":
	// d = dialect.Clickhouse
	// case "vertica":
	// d = dialect.Vertica
	default:
		return fmt.Errorf("%q: unknown dialect", s)
	}
	return nil
}
