package database_test

import (
	"testing"

	"github.com/pressly/goose/v3/database"
)

func TestParseDialectCoverage(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		alias       string
		wantDialect database.Dialect
		wantErr     error
	}{
		{alias: "postgres", wantDialect: database.DialectPostgres},
		{alias: "pgx", wantDialect: database.DialectPostgres},
		{alias: "mysql", wantDialect: database.DialectMySQL},
		{alias: "sqlite", wantDialect: database.DialectSQLite3},
		{alias: "sqlite3", wantDialect: database.DialectSQLite3},
		{alias: "mssql", wantDialect: database.DialectMSSQL},
		{alias: "azuresql", wantDialect: database.DialectMSSQL},
		{alias: "sqlserver", wantDialect: database.DialectMSSQL},
		{alias: "redshift", wantDialect: database.DialectRedshift},
		{alias: "tidb", wantDialect: database.DialectTiDB},
		{alias: "clickhouse", wantDialect: database.DialectClickHouse},
		{alias: "vertica", wantDialect: database.DialectVertica},
		{alias: "ydb", wantDialect: database.DialectYdB},
		{alias: "turso", wantDialect: database.DialectTurso},
		{alias: "starrocks", wantDialect: database.DialectStarrocks},
		{alias: "dsql", wantDialect: database.DialectAuroraDSQL},
		{
			alias:       "bad",
			wantDialect: database.DialectAuroraDSQL, wantErr: database.ErrUnknownDialect,
		},
	} {
		d, err := database.ParseDialect(tc.alias)
		if tc.wantErr != nil {
			if tc.wantErr != err {
				t.Fatalf("%s: want error: %v, got error: %v", tc.alias, tc.wantErr, err)
			}
		} else if tc.wantDialect != d {
			t.Fatalf("%s: want dialect: %v, got dialect: %v", tc.alias, tc.wantDialect, d)
		}
	}
}
