//go:build !no_mysql
// +build !no_mysql

package main

import (
	_ "github.com/go-sql-driver/mysql"
)
