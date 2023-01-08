package goose

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/table"
	"github.com/ydb-platform/ydb-go-sdk/v3/table/types"
)

// SQLDialect abstracts the details of specific SQL dialects
// for goose's few SQL specific statements
type SQLDialect interface {
	createVersionTableSQL() string // sql string to create the db version table
	insertVersionSQL() string      // sql string to insert the initial version table row
	deleteVersionSQL() string      // sql string to delete version
	migrationSQL() string          // sql string to retrieve migrations
	dbVersionQuery(db *sql.DB) (*sql.Rows, error)
}

var dialect SQLDialect = &PostgresDialect{}

// GetDialect gets the SQLDialect
func GetDialect() SQLDialect {
	return dialect
}

// SetDialect sets the SQLDialect
func SetDialect(d string) error {
	switch d {
	case "postgres", "pgx":
		dialect = &PostgresDialect{}
	case "mysql":
		dialect = &MySQLDialect{}
	case "sqlite3", "sqlite":
		dialect = &Sqlite3Dialect{}
	case "mssql":
		dialect = &SqlServerDialect{}
	case "redshift":
		dialect = &RedshiftDialect{}
	case "tidb":
		dialect = &TiDBDialect{}
	case "clickhouse":
		dialect = &ClickHouseDialect{}
	case "vertica":
		dialect = &VerticaDialect{}
	case "ydb":
		dialect = &YDBDialect{}
	default:
		return fmt.Errorf("%q: unknown dialect", d)
	}

	return nil
}

////////////////////////////
// Postgres
////////////////////////////

// PostgresDialect struct.
type PostgresDialect struct{}

func (pg PostgresDialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
            	id serial NOT NULL,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`, TableName())
}

func (pg PostgresDialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES ($1, $2);", TableName())
}

func (pg PostgresDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied from %s ORDER BY id DESC", TableName()))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (m PostgresDialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id=$1 ORDER BY tstamp DESC LIMIT 1", TableName())
}

func (pg PostgresDialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=$1;", TableName())
}

////////////////////////////
// MySQL
////////////////////////////

// MySQLDialect struct.
type MySQLDialect struct{}

func (m MySQLDialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
                id serial NOT NULL,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`, TableName())
}

func (m MySQLDialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES (?, ?);", TableName())
}

func (m MySQLDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied from %s ORDER BY id DESC", TableName()))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (m MySQLDialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id=? ORDER BY tstamp DESC LIMIT 1", TableName())
}

func (m MySQLDialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=?;", TableName())
}

////////////////////////////
// MSSQL
////////////////////////////

// SqlServerDialect struct.
type SqlServerDialect struct{}

func (m SqlServerDialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
                id INT NOT NULL IDENTITY(1,1) PRIMARY KEY,
                version_id BIGINT NOT NULL,
                is_applied BIT NOT NULL,
                tstamp DATETIME NULL DEFAULT CURRENT_TIMESTAMP
            );`, TableName())
}

func (m SqlServerDialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES (@p1, @p2);", TableName())
}

func (m SqlServerDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied FROM %s ORDER BY id DESC", TableName()))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (m SqlServerDialect) migrationSQL() string {
	const tpl = `
WITH Migrations AS
(
    SELECT tstamp, is_applied,
    ROW_NUMBER() OVER (ORDER BY tstamp) AS 'RowNumber'
    FROM %s
	WHERE version_id=@p1
)
SELECT tstamp, is_applied
FROM Migrations
WHERE RowNumber BETWEEN 1 AND 2
ORDER BY tstamp DESC
`
	return fmt.Sprintf(tpl, TableName())
}

func (m SqlServerDialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=@p1;", TableName())
}

////////////////////////////
// sqlite3
////////////////////////////

// Sqlite3Dialect struct.
type Sqlite3Dialect struct{}

func (m Sqlite3Dialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                version_id INTEGER NOT NULL,
                is_applied INTEGER NOT NULL,
                tstamp TIMESTAMP DEFAULT (datetime('now'))
            );`, TableName())
}

func (m Sqlite3Dialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES (?, ?);", TableName())
}

func (m Sqlite3Dialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied from %s ORDER BY id DESC", TableName()))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (m Sqlite3Dialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id=? ORDER BY tstamp DESC LIMIT 1", TableName())
}

func (m Sqlite3Dialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=?;", TableName())
}

////////////////////////////
// Redshift
////////////////////////////

// RedshiftDialect struct.
type RedshiftDialect struct{}

func (rs RedshiftDialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
            	id integer NOT NULL identity(1, 1),
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default sysdate,
                PRIMARY KEY(id)
            );`, TableName())
}

func (rs RedshiftDialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES ($1, $2);", TableName())
}

func (rs RedshiftDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied from %s ORDER BY id DESC", TableName()))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (m RedshiftDialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id=$1 ORDER BY tstamp DESC LIMIT 1", TableName())
}

func (rs RedshiftDialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=$1;", TableName())
}

////////////////////////////
// TiDB
////////////////////////////

// TiDBDialect struct.
type TiDBDialect struct{}

func (m TiDBDialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
                id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT UNIQUE,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`, TableName())
}

func (m TiDBDialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES (?, ?);", TableName())
}

func (m TiDBDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied from %s ORDER BY id DESC", TableName()))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (m TiDBDialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id=? ORDER BY tstamp DESC LIMIT 1", TableName())
}

func (m TiDBDialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=?;", TableName())
}

////////////////////////////
// ClickHouse
////////////////////////////

// ClickHouseDialect struct.
type ClickHouseDialect struct{}

func (m ClickHouseDialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
      version_id Int64,
      is_applied UInt8,
      date Date default now(),
      tstamp DateTime default now()
    )
	ENGINE = MergeTree()
	  ORDER BY (date)`, TableName())
}

func (m ClickHouseDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied FROM %s ORDER BY version_id DESC", TableName()))
	if err != nil {
		return nil, err
	}
	return rows, err
}

func (m ClickHouseDialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES ($1, $2)", TableName())
}

func (m ClickHouseDialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id = $1 ORDER BY tstamp DESC LIMIT 1", TableName())
}

func (m ClickHouseDialect) deleteVersionSQL() string {
	return fmt.Sprintf("ALTER TABLE %s DELETE WHERE version_id = $1", TableName())
}

////////////////////////////
// Vertica
////////////////////////////

// VerticaDialect struct.
type VerticaDialect struct{}

func (v VerticaDialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
                id identity(1,1) NOT NULL,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`, TableName())
}

func (v VerticaDialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES (?, ?);", TableName())
}

func (v VerticaDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied from %s ORDER BY id DESC", TableName()))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (m VerticaDialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id=? ORDER BY tstamp DESC LIMIT 1", TableName())
}

func (v VerticaDialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=?;", TableName())
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

type execer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
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
