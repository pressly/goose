// +build no_mysql

package main

import (
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/ziutek/mymysql/godrv"
)

func normalizeDBString(driver string, str string, tls bool) string {
	return str
}

func registerTLSConfig(_ string) error {
	return nil
}
