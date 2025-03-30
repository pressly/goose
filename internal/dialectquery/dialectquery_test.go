package dialectquery_test

import (
	"github.com/pressly/goose/v4"
	"github.com/pressly/goose/v4/internal/dialect"
	"github.com/pressly/goose/v4/internal/dialectquery"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCheckDialects(t *testing.T) {
	tests := []struct {
		name                 string
		querier              dialectquery.Querier
		wantDialect          dialect.Dialect
		wantSqlDeleteVersion string
	}{
		{
			name:                 "clickhouse",
			querier:              &dialectquery.Clickhouse{},
			wantDialect:          dialect.Clickhouse,
			wantSqlDeleteVersion: `ALTER TABLE goose_db_version DELETE WHERE version_id = $1 SETTINGS mutations_sync = 2`,
		},
		{
			name:                 "mysql",
			querier:              &dialectquery.Mysql{},
			wantDialect:          dialect.Mysql,
			wantSqlDeleteVersion: `DELETE FROM goose_db_version WHERE version_id=?`,
		},
		{
			name:                 "postgres",
			querier:              &dialectquery.Postgres{},
			wantDialect:          dialect.Postgres,
			wantSqlDeleteVersion: `DELETE FROM goose_db_version WHERE version_id=$1`,
		},
		{
			name:                 "redshift",
			querier:              &dialectquery.Redshift{},
			wantDialect:          dialect.Redshift,
			wantSqlDeleteVersion: `DELETE FROM goose_db_version WHERE version_id=$1`,
		},
		{
			name:                 "sqlite",
			querier:              &dialectquery.Sqlite3{},
			wantDialect:          dialect.Sqlite3,
			wantSqlDeleteVersion: `DELETE FROM goose_db_version WHERE version_id=?`,
		},
		{
			name:                 "sqlserver",
			querier:              &dialectquery.Sqlserver{},
			wantDialect:          dialect.Sqlserver,
			wantSqlDeleteVersion: `DELETE FROM goose_db_version WHERE version_id=@p1`,
		},
		{
			name:                 "starrocks",
			querier:              &dialectquery.Starrocks{},
			wantDialect:          dialect.Starrocks,
			wantSqlDeleteVersion: `DELETE FROM goose_db_version WHERE version_id=?`,
		},
		{
			name:                 "tidb",
			querier:              &dialectquery.Tidb{},
			wantDialect:          dialect.Tidb,
			wantSqlDeleteVersion: `DELETE FROM goose_db_version WHERE version_id=?`,
		},
		{
			name:                 "turso",
			querier:              &dialectquery.Turso{},
			wantDialect:          dialect.Turso,
			wantSqlDeleteVersion: `DELETE FROM goose_db_version WHERE version_id=?`,
		},
		{
			name:                 "vertica",
			querier:              &dialectquery.Vertica{},
			wantDialect:          dialect.Vertica,
			wantSqlDeleteVersion: `DELETE FROM goose_db_version WHERE version_id=?`,
		},
		{
			name:                 "ydb",
			querier:              &dialectquery.Ydb{},
			wantDialect:          dialect.Ydb,
			wantSqlDeleteVersion: `DELETE FROM goose_db_version WHERE version_id = $1`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.wantDialect, test.querier.GetDialect())
			require.Equal(t, test.wantSqlDeleteVersion, test.querier.DeleteVersion(goose.DefaultTablename))
		})
	}
}
