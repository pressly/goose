package goose

import (
	"context"
	"database/sql"
	"fmt"
	mssql "github.com/denisenkom/go-mssqldb"

	"github.com/pkg/errors"
)

// SQLDialect abstracts the details of specific SQL dialects
// for goose's few SQL specific statements
type SQLDialect interface {
	createVersionTableSQL() string // sql string to create the db version table
	insertVersionSQL() string      // sql string to insert the initial version table row
	deleteVersionSQL() string      // sql string to delete version
	migrationSQL() string          // sql string to retrieve migrations
	dbVersionQuery(db *sql.DB) (*sql.Rows, error)
	lock(db *sql.DB)error
	unlock(db *sql.DB)error
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
	default:
		return fmt.Errorf("%q: unknown dialect", d)
	}

	return nil
}

////////////////////////////
// Postgres
////////////////////////////

// PostgresDialect struct.
type PostgresDialect struct{
	isLocked bool
}

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

func(pg PostgresDialect)lock(db *sql.DB)error{
	if pg.isLocked {
		return errors.New(TableName()+" is locked")
	}

	aid, err := generateAdvisoryLockId(TableName())
	if err != nil {
		return err
	}

	// This will wait indefinitely until the lock can be acquired.
	query := `SELECT pg_advisory_lock($1)`
	if _, err := db.ExecContext(context.Background(), query, aid); err != nil {
		return fmt.Errorf("try lock failed, err: %s",err.Error())
	}

	pg.isLocked = true
	return nil
}

func(pg PostgresDialect)unlock(db *sql.DB)error{
	if !pg.isLocked {
		return nil
	}

	aid, err := generateAdvisoryLockId(TableName())
	if err != nil {
		return err
	}

	query := `SELECT pg_advisory_unlock($1)`
	if _, err := db.ExecContext(context.Background(), query, aid); err != nil {
		return fmt.Errorf("try unlock failed, err: %s",err.Error())
	}
	pg.isLocked = false
	return nil
}

////////////////////////////
// MySQL
////////////////////////////

// MySQLDialect struct.
type MySQLDialect struct{
	isLocked bool
}

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

func(m MySQLDialect)lock(db *sql.DB)error{
	if m.isLocked {
		return errors.New(TableName() + " is locked")
	}

	aid, err := generateAdvisoryLockId(TableName())
	if err != nil {
		return err
	}

	query := "SELECT GET_LOCK(?, 10)"
	var success bool
	if err := db.QueryRowContext(context.Background(), query, aid).Scan(&success); err != nil {
		return fmt.Errorf("try lock failed, err: %s",err.Error())
	}

	if success {
		m.isLocked = true
		return nil
	}

	return fmt.Errorf("try lock failed, no error")
}

func(m MySQLDialect)unlock(db *sql.DB)error{
	if !m.isLocked {
		return nil
	}

	aid, err := generateAdvisoryLockId(TableName())
	if err != nil {
		return err
	}

	query := `SELECT RELEASE_LOCK(?)`
	if _, err := db.ExecContext(context.Background(), query, aid); err != nil {
		return fmt.Errorf("try unlock failed, err: %s",err.Error())
	}

	// NOTE: RELEASE_LOCK could return NULL or (or 0 if the code is changed),
	// in which case isLocked should be true until the timeout expires -- synchronizing
	// these states is likely not worth trying to do; reconsider the necessity of isLocked.

	m.isLocked = false
	return nil
}

////////////////////////////
// MSSQL
////////////////////////////

// SqlServerDialect struct.
type SqlServerDialect struct{
	isLocked bool
}

var lockErrorMap = map[mssql.ReturnStatus]string{
	-1:   "The lock request timed out.",
	-2:   "The lock request was canceled.",
	-3:   "The lock request was chosen as a deadlock victim.",
	-999: "Parameter validation or other call error.",
}

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

func(m SqlServerDialect)lock(db *sql.DB)error{
	if m.isLocked {
		return errors.New(TableName() + " is locked")
	}

	aid, err := generateAdvisoryLockId(TableName())
	if err != nil {
		return err
	}

	// This will either obtain the lock immediately and return true,
	// or return false if the lock cannot be acquired immediately.
	// MS Docs: sp_getapplock: https://docs.microsoft.com/en-us/sql/relational-databases/system-stored-procedures/sp-getapplock-transact-sql?view=sql-server-2017
	query := `EXEC sp_getapplock @Resource = @p1, @LockMode = 'Update', @LockOwner = 'Session', @LockTimeout = 0`

	var status mssql.ReturnStatus
	if _, err = db.ExecContext(context.Background(), query, aid, &status); err == nil && status > -1 {
		m.isLocked = true
		return nil
	} else if err != nil {
		fmt.Errorf("try lock failed, err: %s",err.Error())
	} else {
		return fmt.Errorf("try lock failed, err: %s", lockErrorMap[status])
	}

	return nil
}

func(m SqlServerDialect)unlock(db *sql.DB)error{
	if !m.isLocked {
		return nil
	}

	aid, err := generateAdvisoryLockId(TableName())
	if err != nil {
		return err
	}

	// MS Docs: sp_releaseapplock: https://docs.microsoft.com/en-us/sql/relational-databases/system-stored-procedures/sp-releaseapplock-transact-sql?view=sql-server-2017
	query := `EXEC sp_releaseapplock @Resource = @p1, @LockOwner = 'Session'`
	if _, err := db.ExecContext(context.Background(), query, aid); err != nil {
		return fmt.Errorf("try unlock failed, err: %s",err.Error())
	}
	m.isLocked = false

	return nil
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

func(m Sqlite3Dialect)lock(db *sql.DB)error{
	return nil
}

func(m Sqlite3Dialect)unlock(db *sql.DB)error{
	return nil
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

func(rs RedshiftDialect)lock(db *sql.DB)error{
	return nil
}

func(rs RedshiftDialect)unlock(db *sql.DB)error{
	return nil
}


////////////////////////////
// TiDB
////////////////////////////

// TiDBDialect struct.
type TiDBDialect struct{
	isLocked bool
}

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

func(m TiDBDialect)lock(db *sql.DB)error{
	if m.isLocked {
		return errors.New(TableName() + " is locked")
	}

	aid, err := generateAdvisoryLockId(TableName())
	if err != nil {
		return err
	}

	query := "SELECT GET_LOCK(?, 10)"
	var success bool
	if err := db.QueryRowContext(context.Background(), query, aid).Scan(&success); err != nil {
		return fmt.Errorf("try lock failed, err: %s",err.Error())
	}

	if success {
		m.isLocked = true
		return nil
	}

	return fmt.Errorf("try lock failed, no error")
}

func(m TiDBDialect)unlock(db *sql.DB)error{
	if !m.isLocked {
		return nil
	}

	aid, err := generateAdvisoryLockId(TableName())
	if err != nil {
		return err
	}

	query := `SELECT RELEASE_LOCK(?)`
	if _, err := db.ExecContext(context.Background(), query, aid); err != nil {
		return fmt.Errorf("try unlock failed, err: %s",err.Error())
	}

	// NOTE: RELEASE_LOCK could return NULL or (or 0 if the code is changed),
	// in which case isLocked should be true until the timeout expires -- synchronizing
	// these states is likely not worth trying to do; reconsider the necessity of isLocked.

	m.isLocked = false
	return nil
}

////////////////////////////
// ClickHouse
////////////////////////////

// ClickHouseDialect struct.
type ClickHouseDialect struct{}

func (m ClickHouseDialect) createVersionTableSQL() string {
	return `
    CREATE TABLE goose_db_version (
      version_id Int64,
      is_applied UInt8,
      date Date default now(),
      tstamp DateTime default now()
    ) Engine = MergeTree(date, (date), 8192)
	`
}

func (m ClickHouseDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("SELECT version_id, is_applied FROM %s ORDER BY tstamp DESC LIMIT 1", TableName()))
	if err != nil {
		return nil, err
	}
	return rows, err
}

func (m ClickHouseDialect) insertVersionSQL() string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES (?, ?)", TableName())
}

func (m ClickHouseDialect) migrationSQL() string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id = ? ORDER BY tstamp DESC LIMIT 1", TableName())
}

func (m ClickHouseDialect) deleteVersionSQL() string {
	return fmt.Sprintf("ALTER TABLE %s DELETE WHERE version_id = ?", TableName())
}

func(m ClickHouseDialect)lock(db *sql.DB)error{
	return nil
}

func(m ClickHouseDialect)unlock(db *sql.DB)error{
	return nil
}