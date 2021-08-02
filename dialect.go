package goose

import (
	"context"
	"database/sql"
	"fmt"
)

// SQLDialect abstracts the details of specific SQL dialects
// for goose's few SQL specific statements
type SQLDialect interface {
	createVersionTableSQL() []string                            // sql string to create the db version table
	insertVersion(tx execer, version int64, applied bool) error // sql string to insert the initial version table row
	deleteVersionSQL() string                                   // sql string to delete version
	migrationSQL() string                                       // sql string to retrieve migrations
	dbVersionQuery(db *sql.DB) (*sql.Rows, error)
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
}

var dialect SQLDialect = &PostgresDialect{}

// GetDialect gets the SQLDialect
func GetDialect() SQLDialect {
	return dialect
}

// SetDialect sets the SQLDialect
func SetDialect(d string) error {
	switch d {
	case "postgres":
		dialect = &PostgresDialect{}
	case "mysql":
		dialect = &MySQLDialect{}
	case "sqlite3":
		dialect = &Sqlite3Dialect{}
	case "mssql":
		dialect = &SqlServerDialect{}
	case "redshift":
		dialect = &RedshiftDialect{}
	case "tidb":
		dialect = &TiDBDialect{}
	case "clickhouse":
		dialect = &ClickHouseDialect{}
	case "oracle":
		fallthrough
	case "godror":
		dialect = &OracleDialect{}
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

func (pg PostgresDialect) createVersionTableSQL() []string {
	return []string{fmt.Sprintf(`CREATE TABLE %s (
            	id serial NOT NULL,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`, TableName())}
}

func (pg PostgresDialect) insertVersion(tx execer, version int64, applied bool) error {
	const stmt = "INSERT INTO %s (version_id, is_applied) VALUES ($1, $2)"
	_, err := tx.Exec(fmt.Sprintf(stmt, TableName()), version, applied)
	return err
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

func (m MySQLDialect) createVersionTableSQL() []string {
	return []string{fmt.Sprintf(`CREATE TABLE %s (
                id serial NOT NULL,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`, TableName())}
}

func (m MySQLDialect) insertVersion(tx execer, version int64, applied bool) error {
	const stmt = "INSERT INTO %s (version_id, is_applied) VALUES (?, ?);"
	_, err := tx.Exec(fmt.Sprintf(stmt, TableName()), version, applied)
	return err
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

func (m SqlServerDialect) createVersionTableSQL() []string {
	return []string{fmt.Sprintf(`CREATE TABLE %s (
                id INT NOT NULL IDENTITY(1,1) PRIMARY KEY,
                version_id BIGINT NOT NULL,
                is_applied BIT NOT NULL,
                tstamp DATETIME NULL DEFAULT CURRENT_TIMESTAMP
            );`, TableName())}
}

func (m SqlServerDialect) insertVersion(tx execer, version int64, applied bool) error {
	const stmt = "INSERT INTO %s (version_id, is_applied) VALUES (@p1, @p2);"
	_, err := tx.Exec(fmt.Sprintf(stmt, TableName()), version, applied)
	return err
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

func (m Sqlite3Dialect) createVersionTableSQL() []string {
	return []string{fmt.Sprintf(`CREATE TABLE %s (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                version_id INTEGER NOT NULL,
                is_applied INTEGER NOT NULL,
                tstamp TIMESTAMP DEFAULT (datetime('now'))
            );`, TableName())}
}

func (m Sqlite3Dialect) insertVersion(tx execer, version int64, applied bool) error {
	const stmt = "INSERT INTO %s (version_id, is_applied) VALUES (?, ?)"
	_, err := tx.Exec(fmt.Sprintf(stmt, TableName()), version, applied)
	return err
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

func (rs RedshiftDialect) createVersionTableSQL() []string {
	return []string{fmt.Sprintf(`CREATE TABLE %s (
            	id integer NOT NULL identity(1, 1),
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default sysdate,
                PRIMARY KEY(id)
            );`, TableName())}
}

func (rs RedshiftDialect) insertVersion(tx execer, version int64, applied bool) error {
	const stmt = "INSERT INTO %s (version_id, is_applied) VALUES ($1, $2)"
	_, err := tx.Exec(fmt.Sprintf(stmt, TableName()), version, applied)
	return err
}

func (rs RedshiftDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied from %s ORDER BY id DESC", TableName()))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (rs RedshiftDialect) migrationSQL() string {
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

func (m TiDBDialect) createVersionTableSQL() []string {
	return []string{fmt.Sprintf(`CREATE TABLE %s (
                id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT UNIQUE,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`, TableName())}
}

func (m TiDBDialect) insertVersion(tx execer, version int64, applied bool) error {
	const stmt = "INSERT INTO %s (version_id, is_applied) VALUES (?, ?)"
	_, err := tx.Exec(fmt.Sprintf(stmt, TableName()), version, applied)
	return err
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

func (m ClickHouseDialect) createVersionTableSQL() []string {
	return []string{fmt.Sprintf(`
    CREATE TABLE %s (
      version_id Int64,
      is_applied UInt8,
      date Date default now(),
      tstamp DateTime default now()
    ) Engine = MergeTree(date, (date), 8192)
	`, TableName())}
}

func (m ClickHouseDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied FROM %s ORDER BY tstamp DESC LIMIT 1", TableName()))
	if err != nil {
		return nil, err
	}
	return rows, err
}

func (m ClickHouseDialect) insertVersion(tx execer, version int64, applied bool) error {
	const stmt = "INSERT INTO %s (version_id, is_applied) VALUES (?, ?)"
	_, err := tx.Exec(fmt.Sprintf(stmt, TableName()), version, applied)
	return err
}

func (m ClickHouseDialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id = ? ORDER BY tstamp DESC LIMIT 1", TableName())
}

func (m ClickHouseDialect) deleteVersionSQL() string {
	return fmt.Sprintf("ALTER TABLE %s DELETE WHERE version_id = ?", TableName())
}

// OracleDialect struct.
type OracleDialect struct{}

func (m OracleDialect) createVersionTableSQL() []string {
	stmts := []string{
		fmt.Sprintf("CREATE SEQUENCE %[1]s_seq START WITH 1", TableName()),
		fmt.Sprintf(`CREATE TABLE %[1]s (
                id NUMBER(10)    DEFAULT %[1]s_seq.nextval NOT NULL,
                version_id NUMBER(10) NOT NULL,
                is_applied CHAR(1) DEFAULT 'N' NOT NULL,
                tstamp timestamp default current_timestamp,
                PRIMARY KEY(id)
            )`, TableName()),
	}
	return stmts
}

func (m OracleDialect) insertVersion(tx execer, version int64, applied bool) error {
	// Oracle does not have a boolean datatype, so using a wrapper booler type.
	// see this example: https://github.com/godror/godror/commit/109c805bf7ad5c92018b72b956cc4ff420104d4a
	const stmt = `INSERT INTO %s (version_id, is_applied) VALUES (:1, :2)`
	_, err := tx.Exec(fmt.Sprintf(stmt, TableName()), version, booler(applied))
	return err
}

func (m OracleDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied from %s ORDER BY id DESC", TableName()))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (m OracleDialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id=:1 ORDER BY tstamp DESC LIMIT 1", TableName())
}

func (m OracleDialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=:1", TableName())
}
