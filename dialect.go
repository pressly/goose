package goose

import (
	"database/sql"
	"fmt"
)

const (
	DialectPostgres   = "postgres"
	DialectSQLite3    = "sqlite3"
	DialectMySQL      = "mysql"
	DialectMSSQL      = "mssql"
	DialectRedShit    = "redshift"
	DialectTiDB       = "tidb"
	DialectClickHouse = "clickhouse"
)

// SQLDialect abstracts the details of specific SQL dialects
// for goose's few SQL specific statements
type SQLDialect interface {
	SetTableName(name string)      // set table name to use for SQL generation
	createVersionTableSQL() string // sql string to create the db version table
	insertVersionSQL() string      // sql string to insert the initial version table row
	deleteVersionSQL() string      // sql string to delete version
	migrationSQL() string          // sql string to retrieve migrations
	dbVersionQuery(db *sql.DB) (*sql.Rows, error)
}

// GetDialect gets the SQLDialect
func GetDialect() SQLDialect {
	return defaultProvider.dialect
}

func SelectDialect(tableName, d string) (SQLDialect, error) {

	base := BaseDialect{TableName: tableName}

	switch d {
	case DialectPostgres, "pgx":
		return &PostgresDialect{base}, nil
	case DialectMySQL:
		return &MySQLDialect{base}, nil
	case DialectSQLite3, "sqlite":
		return &Sqlite3Dialect{base}, nil
	case DialectMSSQL:
		return &SqlServerDialect{base}, nil
	case DialectRedShit:
		return &RedshiftDialect{base}, nil
	case DialectTiDB:
		return &TiDBDialect{base}, nil
	case DialectClickHouse:
		return &ClickHouseDialect{base}, nil
	default:
		return nil, fmt.Errorf("%q: unknown dialect", d)
	}
}

// Dialect returns the SQLDialect of the provider
func (p *Provider) Dialect() SQLDialect { return p.dialect }

// SetDialect sets the SQLDialect
func SetDialect(d string) error {
	return defaultProvider.SetDialect(d)
}

// SetDialect sets the SQLDialect
func (p *Provider) SetDialect(d string) error {
	dialect, err := SelectDialect(p.tableName, d)
	if err != nil {
		return err
	}
	p.dialect = dialect
	return nil
}

// BaseDialect struct.
type BaseDialect struct {
	TableName string
}

func (bd *BaseDialect) SetTableName(name string) {
	bd.TableName = name
}

////////////////////////////
// Postgres
////////////////////////////

// PostgresDialect struct.
type PostgresDialect struct{ BaseDialect }

func (d PostgresDialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
            	id serial NOT NULL,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`, d.TableName)
}

func (d PostgresDialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES ($1, $2);", d.TableName)
}

func (d PostgresDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied from %s ORDER BY id DESC", d.TableName))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (d PostgresDialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id=$1 ORDER BY tstamp DESC LIMIT 1", d.TableName)
}

func (d PostgresDialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=$1;", d.TableName)
}

////////////////////////////
// MySQL
////////////////////////////

// MySQLDialect struct.
type MySQLDialect struct{ BaseDialect }

func (d MySQLDialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
                id serial NOT NULL,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`, d.TableName)
}

func (d MySQLDialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES (?, ?);", d.TableName)
}

func (d MySQLDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied from %s ORDER BY id DESC", d.TableName))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (d MySQLDialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id=? ORDER BY tstamp DESC LIMIT 1", d.TableName)
}

func (d MySQLDialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=?;", d.TableName)
}

////////////////////////////
// MSSQL
////////////////////////////

// SqlServerDialect struct.
type SqlServerDialect struct{ BaseDialect }

func (d SqlServerDialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
                id INT NOT NULL IDENTITY(1,1) PRIMARY KEY,
                version_id BIGINT NOT NULL,
                is_applied BIT NOT NULL,
                tstamp DATETIME NULL DEFAULT CURRENT_TIMESTAMP
            );`, d.TableName)
}

func (d SqlServerDialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES (@p1, @p2);", d.TableName)
}

func (d SqlServerDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied FROM %s ORDER BY id DESC", d.TableName))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (d SqlServerDialect) migrationSQL() string {
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
	return fmt.Sprintf(tpl, d.TableName)
}

func (d SqlServerDialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=@p1;", d.TableName)
}

////////////////////////////
// sqlite3
////////////////////////////

// Sqlite3Dialect struct.
type Sqlite3Dialect struct{ BaseDialect }

func (d Sqlite3Dialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                version_id INTEGER NOT NULL,
                is_applied INTEGER NOT NULL,
                tstamp TIMESTAMP DEFAULT (datetime('now'))
            );`, d.TableName)
}

func (d Sqlite3Dialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES (?, ?);", d.TableName)
}

func (d Sqlite3Dialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied from %s ORDER BY id DESC", d.TableName))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (d Sqlite3Dialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id=? ORDER BY tstamp DESC LIMIT 1", d.TableName)
}

func (d Sqlite3Dialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=?;", d.TableName)
}

////////////////////////////
// Redshift
////////////////////////////

// RedshiftDialect struct.
type RedshiftDialect struct{ BaseDialect }

func (d RedshiftDialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
            	id integer NOT NULL identity(1, 1),
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default sysdate,
                PRIMARY KEY(id)
            );`, d.TableName)
}

func (d RedshiftDialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES ($1, $2);", d.TableName)
}

func (d RedshiftDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied from %s ORDER BY id DESC", d.TableName))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (d RedshiftDialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id=$1 ORDER BY tstamp DESC LIMIT 1", d.TableName)
}

func (d RedshiftDialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=$1;", d.TableName)
}

////////////////////////////
// TiDB
////////////////////////////

// TiDBDialect struct.
type TiDBDialect struct{ BaseDialect }

func (d TiDBDialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
                id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT UNIQUE,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`, d.TableName)
}

func (d TiDBDialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES (?, ?);", d.TableName)
}

func (d TiDBDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied from %s ORDER BY id DESC", d.TableName))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (d TiDBDialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id=? ORDER BY tstamp DESC LIMIT 1", d.TableName)
}

func (d TiDBDialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=?;", d.TableName)
}

////////////////////////////
// ClickHouse
////////////////////////////

// ClickHouseDialect struct.
type ClickHouseDialect struct{ BaseDialect }

func (d ClickHouseDialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
      version_id Int64,
      is_applied UInt8,
      date Date default now(),
      tstamp DateTime default now()
    ) Engine = MergeTree(date, (date), 8192)`, d.TableName)
}

func (d ClickHouseDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied FROM %s ORDER BY tstamp DESC", d.TableName))
	if err != nil {
		return nil, err
	}
	return rows, err
}

func (d ClickHouseDialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES ($1, $2)", d.TableName)
}

func (d ClickHouseDialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id = $1 ORDER BY tstamp DESC LIMIT 1", d.TableName)
}

func (d ClickHouseDialect) deleteVersionSQL() string {
	return fmt.Sprintf("ALTER TABLE %s DELETE WHERE version_id = $1", d.TableName)
}
