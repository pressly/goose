package cli

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v4"
	"github.com/xo/dburl"
)

// gooseDrivers maps dialects to the driver names used by the goose CLI.
//
// See the ./cmd/goose directory for the driver imports, which are optionally conditionally compiled
// based on build tags.
var gooseDrivers = map[goose.Dialect]string{
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
	dialect, err := resolveDialect(dbURL.Unaliased)
	if err != nil {
		return nil, "", fmt.Errorf("failed to resolve dialect: %w", err)
	}
	// The driver name is used by the goose CLI to open the database connection. It is specific to
	// the goose CLI and the driver imports in ./cmd/goose.
	driverName, ok := gooseDrivers[dialect]
	if !ok {
		return nil, "", fmt.Errorf("unsupported resolved database dialect: %s", dialect)
	}
	db, err := sql.Open(driverName, dbURL.DSN)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open connection: %w", err)
	}
	return db, dialect, nil
}

// resolveDialect returns the dialect for the given string.
//
// The string can be a dialect name or a driver name. We derive the dialect from the DSN protocol
// scheme.
//
// See: https://github.com/xo/dburl#protocol-schemes-and-aliases
func resolveDialect(s string) (goose.Dialect, error) {
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
	default:
		return "", fmt.Errorf("unknown dialect: %q", s)
	}
}
