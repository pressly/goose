package goose

import (
	"database/sql"
	"fmt"
)

// SqlDialect abstracts the details of specific SQL dialects
// for goose's few SQL specific statements
type SqlDialect interface {
	createVersionTableSql(name string) string // sql string to create the goose_db_version table
	insertVersionSql(name string) string      // sql string to insert the initial version table row
	dbVersionQuery(db *sql.DB, name string) (*sql.Rows, error)
}

func (c *Client) GetDialect() SqlDialect {
	return c.Dialect
}

func GetDialect() SqlDialect {
	return globalGoose.GetDialect()
}

func (c *Client) SetDialect(d string) error {
	switch d {
	case "postgres":
		c.Dialect = &PostgresDialect{}
	case "mysql":
		c.Dialect = &MySqlDialect{}
	case "sqlite3":
		c.Dialect = &Sqlite3Dialect{}
	default:
		return fmt.Errorf("%q: unknown dialect", d)
	}

	return nil
}

func SetDialect(d string) error {
	return globalGoose.SetDialect(d)
}

////////////////////////////
// Postgres
////////////////////////////

type PostgresDialect struct{}

func (pg PostgresDialect) createVersionTableSql(name string) string {
	return `CREATE TABLE ` + name + ` (
            	id serial NOT NULL,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`
}

func (pg PostgresDialect) insertVersionSql(name string) string {
	return "INSERT INTO " + name + " (version_id, is_applied) VALUES ($1, $2);"
}

func (pg PostgresDialect) dbVersionQuery(db *sql.DB, name string) (*sql.Rows, error) {
	rows, err := db.Query("SELECT version_id, is_applied from " + name + " ORDER BY id DESC")
	if err != nil {
		return nil, err
	}

	return rows, err
}

////////////////////////////
// MySQL
////////////////////////////

type MySqlDialect struct{}

func (m MySqlDialect) createVersionTableSql(name string) string {
	return `CREATE TABLE ` + name + ` (
                id serial NOT NULL,
                version_id bigint NOT NULL,
                is_applied boolean NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(id)
            );`
}

func (m MySqlDialect) insertVersionSql(name string) string {
	return "INSERT INTO " + name + " (version_id, is_applied) VALUES (?, ?);"
}

func (m MySqlDialect) dbVersionQuery(db *sql.DB, name string) (*sql.Rows, error) {
	rows, err := db.Query("SELECT version_id, is_applied from " + name + " ORDER BY id DESC")
	if err != nil {
		return nil, err
	}

	return rows, err
}

////////////////////////////
// sqlite3
////////////////////////////

type Sqlite3Dialect struct{}

func (m Sqlite3Dialect) createVersionTableSql(name string) string {
	return `CREATE TABLE ` + name + ` (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                version_id INTEGER NOT NULL,
                is_applied INTEGER NOT NULL,
                tstamp TIMESTAMP DEFAULT (datetime('now'))
            );`
}

func (m Sqlite3Dialect) insertVersionSql(name string) string {
	return "INSERT INTO " + name + " (version_id, is_applied) VALUES (?, ?);"
}

func (m Sqlite3Dialect) dbVersionQuery(db *sql.DB, name string) (*sql.Rows, error) {
	rows, err := db.Query("SELECT version_id, is_applied from " + name + " ORDER BY id DESC")
	if err != nil {
		return nil, err
	}

	return rows, err
}
