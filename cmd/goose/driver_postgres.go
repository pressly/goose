//go:build !no_postgres
// +build !no_postgres

package main

import (
	_ "github.com/jackc/pgx/v4/stdlib"
)
