//go:build !no_libsql
// +build !no_libsql

package main

import (
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)
