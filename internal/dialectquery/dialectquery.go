package dialectquery

import (
	"fmt"
	"github.com/pressly/goose/v3/internal/dialect"
)

// Querier is the interface that wraps the basic methods to create a dialect specific query.
type Querier interface {
	// CreateTable returns the SQL query string to create the db version table.
	CreateTable(tableName string) string
	// TableExists returns the SQL query string to check exist the db version table.
	TableExists(tableName string) string

	// InsertVersion returns the SQL query string to insert a new version into the db version table.
	InsertVersion(tableName string) string

	// DeleteVersion returns the SQL query string to delete a version from the db version table.
	DeleteVersion(tableName string) string

	// GetMigrationByVersion returns the SQL query string to get a single migration by version.
	//
	// The query should return the timestamp and is_applied columns.
	GetMigrationByVersion(tableName string) string

	// ListMigrations returns the SQL query string to list all migrations in descending order by id.
	//
	// The query should return the version_id and is_applied columns.
	ListMigrations(tableName string) string

	// GetLatestVersion returns the SQL query string to get the last version_id from the db version
	// table. Returns a nullable int64 value.
	GetLatestVersion(tableName string) string
}

func LookupQuerier(d dialect.Dialect) (Querier, error) {
	lookup := map[dialect.Dialect]Querier{
		dialect.Clickhouse: &Clickhouse{},
		dialect.Sqlserver:  &Sqlserver{},
		dialect.Mysql:      &Mysql{},
		dialect.Postgres:   &Postgres{},
		dialect.Redshift:   &Redshift{},
		dialect.Sqlite3:    &Sqlite3{},
		dialect.Tidb:       &Tidb{},
		dialect.Vertica:    &Vertica{},
		dialect.Ydb:        &Ydb{},
		dialect.Turso:      &Turso{},
		dialect.Starrocks:  &Starrocks{},
	}
	querier, ok := lookup[d]
	if !ok {
		return nil, fmt.Errorf("unknown dialect: %q", d)
	}

	return querier, nil
}
