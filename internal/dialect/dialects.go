package dialect

import (
	"errors"
	"fmt"
	"strings"
)

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

// GetDialect gets the dialect used in the goose package.
func GetDialect(s string) (Dialect, error) {
	switch strings.ToLower(s) {
	case "postgres", "pgx":
		return Postgres, nil
	case "mysql":
		return Mysql, nil
	case "sqlite3", "sqlite":
		return Sqlite3, nil
	case "mssql", "azuresql", "sqlserver":
		return Mssql, nil
	case "redshift":
		return Redshift, nil
	case "tidb":
		return Tidb, nil
	case "clickhouse":
		return Clickhouse, nil
	case "vertica":
		return Vertica, nil
	case "ydb":
		return Ydb, nil
	case "turso":
		return Turso, nil
	case "starrocks":
		return Starrocks, nil
	default:
		return "", fmt.Errorf("%s: %w", s, ErrUnknownDialect)
	}
}

func (d *Dialect) UnmarshalText(text []byte) error {
	dialect, err := GetDialect(string(text))
	if err != nil {
		return err
	}

	*d = dialect

	return nil
}
