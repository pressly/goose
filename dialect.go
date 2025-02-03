package goose

import (
	"fmt"
	"github.com/pressly/goose/v4/internal/dialect"
	"github.com/pressly/goose/v4/internal/dialectstore"
)

// Dialect is the type of database dialect.
type Dialect = dialect.Dialect

const (
	DialectClickHouse Dialect = dialect.Clickhouse
	// Deprecated: use [DialectSqlserver]
	DialectMSSQL     Dialect = dialect.Sqlserver
	DialectSqlserver Dialect = dialect.Sqlserver
	DialectMySQL     Dialect = dialect.Mysql
	DialectPostgres  Dialect = dialect.Postgres
	DialectRedshift  Dialect = dialect.Redshift
	DialectSQLite3   Dialect = dialect.Sqlite3
	DialectTiDB      Dialect = dialect.Tidb
	DialectVertica   Dialect = dialect.Vertica
	DialectYdB       Dialect = dialect.Ydb
	DialectTurso     Dialect = dialect.Turso
	DialectStarrocks Dialect = dialect.Starrocks
)

// SetDialect sets the dialect to use for the goose package.
func SetDialect[D string | Dialect](d D) error {
	var (
		v   Dialect
		err error
	)

	switch t := any(d).(type) {
	case string:
		v, err = GetDialect(t)
		if err != nil {
			return err
		}
	case Dialect:
		v = t
	}

	store, err = dialectstore.NewStore(v, store.GetTableName())

	return err
}

func GetDialect(s string) (Dialect, error) {
	switch s {
	case "postgres", "pgx":
		return DialectPostgres, nil
	case "mysql":
		return DialectMySQL, nil
	case "sqlite3", "sqlite":
		return DialectSQLite3, nil
	case "mssql", "azuresql", "sqlserver":
		return DialectSqlserver, nil
	case "redshift":
		return DialectRedshift, nil
	case "tidb":
		return DialectTiDB, nil
	case "clickhouse":
		return DialectClickHouse, nil
	case "vertica":
		return DialectVertica, nil
	case "ydb":
		return DialectYdB, nil
	case "turso":
		return DialectTurso, nil
	case "starrocks":
		return DialectStarrocks, nil
	default:
		return "", fmt.Errorf("%q: unknown dialect", s)
	}
}
