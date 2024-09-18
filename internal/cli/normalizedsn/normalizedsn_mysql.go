//go:build !no_mysql
// +build !no_mysql

package normalizedsn

import "github.com/go-sql-driver/mysql"

// DBString parses the dsn used with the mysql driver to always have the parameter `parseTime` set
// to true. This allows internal goose logic to assume that DATETIME/DATE/TIMESTAMP can be scanned
// into the time.Time type.
func DBString(dsn string) (string, error) {
	config, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", err
	}
	config.ParseTime = true
	return config.FormatDSN(), nil
}
