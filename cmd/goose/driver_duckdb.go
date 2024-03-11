//go:build !no_duckdb && !(windows && arm64)
// +build !no_duckdb
// +build !windows !arm64

package main

import (
	_ "github.com/marcboeker/go-duckdb"
)
