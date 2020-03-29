package goose

import (
	"database/sql"
	"fmt"
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

// GetDialect gets the SQLDialect
func GetDialect() SQLDialect { return def.GetDialect() }

// GetDialect gets the SQLDialect
func (in *Instance) GetDialect() SQLDialect {
	return in.dialect
}

// SetDialect sets the SQLDialect
func SetDialect(d string) error { return def.SetDialect(d) }

// SetDialect sets the SQLDialect
func (in *Instance) SetDialect(d string) error {
	switch d {
	case "postgres":
		in.dialect = &PostgresDialect{tableName: in.TableName}
	case "mysql":
		in.dialect = &MySQLDialect{tableName: in.TableName}
	case "sqlite3":
		in.dialect = &Sqlite3Dialect{tableName: in.TableName}
	case "mssql":
		in.dialect = &SqlServerDialect{tableName: in.TableName}
	case "redshift":
		in.dialect = &RedshiftDialect{tableName: in.TableName}
	case "tidb":
		in.dialect = &TiDBDialect{tableName: in.TableName}
	default:
		return fmt.Errorf("%q: unknown dialect", d)
	}

	return nil
}

////////////////////////////
// Postgres
////////////////////////////

// PostgresDialect struct.
type PostgresDialect struct {
	tableName func() string
}

func (pg PostgresDialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
            	id serial NOT NULL,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`, pg.tableName())
}

func (pg PostgresDialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES ($1, $2);", pg.tableName())
}

func (pg PostgresDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied from %s ORDER BY id DESC", pg.tableName()))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (pg PostgresDialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id=$1 ORDER BY tstamp DESC LIMIT 1", pg.tableName())
}

func (pg PostgresDialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=$1;", pg.tableName())
}

////////////////////////////
// MySQL
////////////////////////////

// MySQLDialect struct.
type MySQLDialect struct {
	tableName func() string
}

func (m MySQLDialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
                id serial NOT NULL,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`, m.tableName())
}

func (m MySQLDialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES (?, ?);", m.tableName())
}

func (m MySQLDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied from %s ORDER BY id DESC", m.tableName()))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (m MySQLDialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id=? ORDER BY tstamp DESC LIMIT 1", m.tableName())
}

func (m MySQLDialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=?;", m.tableName())
}

////////////////////////////
// MSSQL
////////////////////////////

// SqlServerDialect struct.
type SqlServerDialect struct {
	tableName func() string
}

func (m SqlServerDialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
                id INT NOT NULL IDENTITY(1,1) PRIMARY KEY,
                version_id BIGINT NOT NULL,
                is_applied BIT NOT NULL,
                tstamp DATETIME NULL DEFAULT CURRENT_TIMESTAMP
            );`, m.tableName())
}

func (m SqlServerDialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES (@p1, @p2);", m.tableName())
}

func (m SqlServerDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied FROM %s ORDER BY id DESC", m.tableName()))
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
	return fmt.Sprintf(tpl, m.tableName())
}

func (m SqlServerDialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=@p1;", m.tableName())
}

////////////////////////////
// sqlite3
////////////////////////////

// Sqlite3Dialect struct.
type Sqlite3Dialect struct {
	tableName func() string
}

func (m Sqlite3Dialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                version_id INTEGER NOT NULL,
                is_applied INTEGER NOT NULL,
                tstamp TIMESTAMP DEFAULT (datetime('now'))
            );`, m.tableName())
}

func (m Sqlite3Dialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES (?, ?);", m.tableName())
}

func (m Sqlite3Dialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied from %s ORDER BY id DESC", m.tableName()))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (m Sqlite3Dialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id=? ORDER BY tstamp DESC LIMIT 1", m.tableName())
}

func (m Sqlite3Dialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=?;", m.tableName())
}

////////////////////////////
// Redshift
////////////////////////////

// RedshiftDialect struct.
type RedshiftDialect struct {
	tableName func() string
}

func (rs RedshiftDialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
            	id integer NOT NULL identity(1, 1),
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default sysdate,
                PRIMARY KEY(id)
            );`, rs.tableName())
}

func (rs RedshiftDialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES ($1, $2);", rs.tableName())
}

func (rs RedshiftDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied from %s ORDER BY id DESC", rs.tableName()))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (rs RedshiftDialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id=$1 ORDER BY tstamp DESC LIMIT 1", rs.tableName())
}

func (rs RedshiftDialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=$1;", rs.tableName())
}

////////////////////////////
// TiDB
////////////////////////////

// TiDBDialect struct.
type TiDBDialect struct {
	tableName func() string
}

func (m TiDBDialect) createVersionTableSQL() string {
	return fmt.Sprintf(`CREATE TABLE %s (
                id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT UNIQUE,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`, m.tableName())
}

func (m TiDBDialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES (?, ?);", m.tableName())
}

func (m TiDBDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied from %s ORDER BY id DESC", m.tableName()))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (m TiDBDialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id=? ORDER BY tstamp DESC LIMIT 1", m.tableName())
}

func (m TiDBDialect) deleteVersionSQL() string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id=?;", m.tableName())
}
