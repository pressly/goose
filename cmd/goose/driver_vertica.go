//go:build !no_vertica
// +build !no_vertica

package main

import (
	_ "github.com/vertica/vertica-sql-go"
)
