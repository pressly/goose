//go:build no_mysql
// +build no_mysql

package main

func normalizeDBString(driver string, str string, certfile string, sslcert string, sslkey string) string {
	return str
}
