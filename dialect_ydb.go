//go:build !no_ydb
// +build !no_ydb

package goose

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/table"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/types"
)

func init() {
	dialects["ydb"] = &YDBDialect{}
}

////////////////////////////
// YDB
////////////////////////////

// YDBDialect struct.
type YDBDialect struct{}

func (m YDBDialect) createVersionTable(db *sql.DB) error {
	ctx := ydb.WithQueryMode(context.Background(), ydb.SchemeQueryMode)

	// create table in transaction is not supported by ydb
	_, err := db.ExecContext(ctx, m.createVersionTableSQL())
	if err != nil {
		return err
	}

	// init version
	err = m.insertVersion(db, 0, true)
	if err != nil {
		return err
	}

	return nil
}

func (m YDBDialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
      version_id Int64,
      is_applied Bool,
      tstamp DateTime,
      PRIMARY KEY(version_id)
    ) `, TableName())
}

func (m YDBDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	// order by version_id instead tstamp because tstamp is not in source list (fails in ydb)
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied FROM %s ORDER BY version_id DESC LIMIT 1", TableName()))
	if err != nil {
		return nil, err
	}
	return rows, err
}

func (m YDBDialect) insertVersion(execer execer, versionID int64, isApplied bool) error {
	_, err := execer.Exec(m.insertVersionSQL(),
		table.ValueParam("versionID", types.Int64Value(versionID)),
		table.ValueParam("isApplied", types.BoolValue(isApplied)),
	)
	return err
}

func (m YDBDialect) insertVersionSQL() string {
	return fmt.Sprintf(`
	DECLARE $versionID AS Int64;
	DECLARE $isApplied AS Bool;

	INSERT INTO %s (version_id, is_applied, tstamp)
	VALUES ($versionID, $isApplied, CurrentUtcDatetime())
`, TableName())
}

func (m YDBDialect) migrationSQL() string {
	return fmt.Sprintf(`
	DECLARE $versionID AS Int64;

	SELECT tstamp, is_applied
	FROM %s
	WHERE version_id = $versionID
	ORDER BY tstamp DESC
	LIMIT 1`, TableName())
}

func (m YDBDialect) migration(db *sql.DB, versionID int64) (*MigrationRecord, error) {
	var row MigrationRecord

	err := db.QueryRow(
		m.migrationSQL(),
		table.ValueParam("versionID", types.Int64Value(versionID)),
	).Scan(&row.TStamp, &row.IsApplied)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return &row, nil
}

func (m YDBDialect) migrateStatement(db *sql.DB, sql string) error {
	ctx := ydb.WithQueryMode(context.Background(), ydb.SchemeQueryMode)

	_, err := db.ExecContext(ctx, sql)
	return err
}

func (m YDBDialect) deleteVersion(execer execer, versionID int64) error {
	_, err := execer.Exec(m.deleteVersionSQL(),
		table.ValueParam("versionID", types.Int64Value(versionID)),
	)
	return err
}

func (m YDBDialect) deleteVersionSQL() string {
	return fmt.Sprintf(`
	DECLARE $versionID AS Int64;

	DELETE FROM %s
	WHERE version_id = $versionID`, TableName())
}
