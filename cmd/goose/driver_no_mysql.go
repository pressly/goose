// +build !mysql

package main

import (
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/ziutek/mymysql/godrv"
)

func normalizeDBString(str string) string {
	return str
}
