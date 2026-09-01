//go:build !no_mssql && !no_azuresql

package main

import (
	_ "github.com/microsoft/go-mssqldb/azuread"
)
