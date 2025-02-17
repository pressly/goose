package goosecli

import (
	"cmp"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mfridman/cli"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/goosecli/normalizedsn"
	"github.com/xo/dburl"
)

const (
	envPrefix = "GOOSE_"
)

var (
	style = lipgloss.NewStyle().Bold(true)
)

func getFlagOrEnv(s *cli.State, name string) (string, error) {
	envName := envPrefix + strings.ToUpper(name)
	val := cmp.Or(
		cli.GetFlag[string](s, name),
		os.Getenv(envName),
	)
	if val == "" {
		return "", fmt.Errorf("must provide --%s flag or set %s environment variable", name, envName)
	}
	return val, nil
}

func getProvider(s *cli.State) (*goose.Provider, error) {
	dir, err := getFlagOrEnv(s, "dir")
	if err != nil {
		return nil, err
	}
	dbstring, err := getFlagOrEnv(s, "dbstring")
	if err != nil {
		return nil, err
	}
	db, dialect, err := openConnection(dbstring)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}
	return goose.NewProvider(dialect, db, os.DirFS(dir))
}

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
		dataSourceName, err = normalizedsn.DBString(dbURL.DSN)
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

func render(s string) string {
	ok, err := strconv.ParseBool(os.Getenv("NO_COLOR"))
	if err == nil && ok {
		return s
	}
	return style.Render(s)
}

func printResults(printer *printer, results []*goose.MigrationResult, useJSON bool) error {
	if useJSON {
		return printer.JSON(toMigrationResult(results))
	}
	table := tableData{
		Headers: []string{"Status", "Migration name", "Duration"},
	}
	for _, result := range results {
		status := "OK"
		if result.Error != nil {
			status = "FAILED"
		}
		if result.Empty {
			status = "EMPTY"
		}
		row := []string{
			status,
			filepath.Base(result.Source.Path),
			truncateDuration(result.Duration).String(),
		}
		table.Rows = append(table.Rows, row)
	}
	return printer.Table(table)
}
