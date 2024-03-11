//go:build !no_duckdb && !windows && !linux && !darwin
// +build !no_duckdb,!windows,!linux,!darwin

package main

import (
	_ "github.com/marcboeker/go-duckdb"
)
