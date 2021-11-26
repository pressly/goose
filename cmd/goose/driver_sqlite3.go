//go:build !no_sqlite3
// +build !no_sqlite3

package main

import (
	_ "modernc.org/sqlite"
)
