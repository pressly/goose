//go:build !no_sqlite3 && !(windows && arm64)

package main

import (
	_ "modernc.org/sqlite"
)
