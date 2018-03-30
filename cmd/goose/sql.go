package main

import (
	"database/sql"
	"fmt"

	"github.com/go-sql-driver/mysql"
)

// normalizeMySQLDSN parses the dsn used with the mysql driver to always have
// the parameter `parseTime` set to true. This allows internal goose logic
// to assume that DATETIME/DATE/TIMESTAMP can be scanned into the time.Time
// type.
func normalizeMySQLDSN(dsn string) (string, error) {
	config, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", err
	}
	config.ParseTime = true
	return config.FormatDSN(), nil
}

func createDBWithDriver(driver string, dbstring string) (*sql.DB, error) {
	switch driver {
	case "postgres", "sqlite3":
		return sql.Open(driver, dbstring)
	case "mysql":
		dsn, err := normalizeMySQLDSN(dbstring)
		if err != nil {
			return nil, err
		}
		return sql.Open(driver, dsn)
	default:
	}
	return nil, fmt.Errorf("unsupported driver %s", driver)
}
