//go:build !no_mssql
// +build !no_mssql

package main

import (
	_ "github.com/microsoft/go-mssqldb"
)
