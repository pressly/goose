// +build no_mysql

package main

func normalizeDBString(driver string, str string, certfile string) string {
	return str
}
