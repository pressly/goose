package goose

import (
	"fmt"
	"strings"

	"github.com/pressly/goose/v3/internal/dialect"
)

// Dialect is the type of database dialect. It is an alias for [dialect.Dialect].
type Dialect = dialect.Dialect

const (
	DialectClickHouse Dialect = dialect.Clickhouse
	DialectMSSQL      Dialect = dialect.Mssql
	DialectMySQL      Dialect = dialect.Mysql
	DialectPostgres   Dialect = dialect.Postgres
	DialectRedshift   Dialect = dialect.Redshift
	DialectSQLite3    Dialect = dialect.Sqlite3
	DialectTiDB       Dialect = dialect.Tidb
	DialectVertica    Dialect = dialect.Vertica
	DialectYdB        Dialect = dialect.Ydb
	DialectTurso      Dialect = dialect.Turso
	DialectStarrocks  Dialect = dialect.Starrocks
)

var ErrUnknownDialect = dialect.ErrUnknownDialect

func init() {
	store, _ = dialect.NewStore(dialect.Postgres)
}

var store dialect.Store

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

	store, err = dialect.NewStore(v)
	return err
}

// GetDialect gets the dialect used in the goose package.
func GetDialect(s string) (Dialect, error) {
	switch strings.ToLower(s) {
	case "postgres", "pgx":
		return DialectPostgres, nil
	case "mysql":
		return DialectMySQL, nil
	case "sqlite3", "sqlite":
		return DialectSQLite3, nil
	case "mssql", "azuresql", "sqlserver":
		return DialectMSSQL, nil
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
		return "", fmt.Errorf("%s: %w", s, ErrUnknownDialect)
	}
}
