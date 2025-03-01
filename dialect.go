package goose

import (
	"github.com/pressly/goose/v4/internal/dialect"
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

var ErrUnknownDialect = dialect.ErrUnknownDialect

// GetDialect gets the dialect used in the goose package.
var GetDialect = dialect.GetDialect

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

	store, err = NewStore(v, store.GetTableName())

	return err
}
