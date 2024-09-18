package cli

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/cli/normalizedsn"
	"github.com/xo/dburl"
)

// dialectToDriverMapping maps dialects to the actual driver names used by the goose CLI.
//
// See the ./cmd/goose directory for driver imports, which are conditionally compiled based on build
// tags. For example, for postgres we use github.com/jackc/pgx/v5/stdlib, and the driver name is
// "pgx". For sqlite3 we use modernc.org/sqlite and the driver name is "sqlite".
var dialectToDriverMapping = map[goose.Dialect]string{
	goose.DialectPostgres:   "pgx",
	goose.DialectRedshift:   "pgx",
	goose.DialectMySQL:      "mysql",
	goose.DialectTiDB:       "mysql",
	goose.DialectSQLite3:    "sqlite",
	goose.DialectMSSQL:      "sqlserver",
	goose.DialectClickHouse: "clickhouse",
	goose.DialectVertica:    "vertica",
}

func openConnection(dbstring string) (*sql.DB, goose.Dialect, error) {
	dbURL, err := dburl.Parse(dbstring)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse DSN: %w", err)
	}
	dialect, err := resolveDialect(dbURL.UnaliasedDriver, dbURL.Scheme)
	if err != nil {
		return nil, "", fmt.Errorf("failed to resolve dialect: %w", err)
	}
	var dataSourceName string
	switch dialect {
	case goose.DialectMySQL:
		dataSourceName, err = normalizedsn.DBString(dataSourceName)
		if err != nil {
			return nil, "", fmt.Errorf("failed to normalize mysql DSN: %w", err)
		}
	default:
		dataSourceName = dbURL.DSN
	}
	driverName, ok := dialectToDriverMapping[dialect]
	if !ok {
		return nil, "", fmt.Errorf("unknown database dialect: %s", dialect)
	}
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open connection: %w", err)
	}
	return db, dialect, nil
}

// resolveDialect returns the dialect for the first string that matches a known dialect alias or
// schema name. If no match is found, an error is returned.
//
// The string can be a schema name or an alias. The aliases are defined by the dburl package for
// common databases. See: https://github.com/xo/dburl#database-schemes-aliases-and-drivers
func resolveDialect(ss ...string) (goose.Dialect, error) {
	for _, s := range ss {
		switch s {
		case "postgres", "pg", "pgx", "postgresql", "pgsql":
			return goose.DialectPostgres, nil
		case "mysql", "my", "mariadb", "maria", "percona", "aurora":
			return goose.DialectMySQL, nil
		case "sqlite", "sqlite3":
			return goose.DialectSQLite3, nil
		case "sqlserver", "ms", "mssql", "azuresql":
			return goose.DialectMSSQL, nil
		case "redshift", "rs":
			return goose.DialectRedshift, nil
		case "tidb", "ti":
			return goose.DialectTiDB, nil
		case "clickhouse", "ch":
			return goose.DialectClickHouse, nil
		case "vertica", "ve":
			return goose.DialectVertica, nil
		}
	}
	return "", fmt.Errorf("failed to resolve scheme names or aliases to a dialect: %q", strings.Join(ss, ","))
}
