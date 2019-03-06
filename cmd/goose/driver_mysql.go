// +build mysql

package main

import (
	"log"

	"github.com/go-sql-driver/mysql"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/ziutek/mymysql/godrv"
)

// normalizeMySQLDSN parses the dsn used with the mysql driver to always have
// the parameter `parseTime` set to true. This allows internal goose logic
// to assume that DATETIME/DATE/TIMESTAMP can be scanned into the time.Time
// type.
func normalizeDBString(str string) string {
	var err error
	str, err = normalizeMySQLDSN(dns string)
	if err != nil {
		log.Fatalf("failed to normalize MySQL connection string: %v", err)
	}
}

func normalizeMySQLDSN(dns string) (string, error) {
    config, err := mysql.ParseDSN(dsn)
    if err != nil {
        return "", err
    }
    config.ParseTime = true
    return config.FormatDSN(), nil
}
