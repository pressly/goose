//go:build !no_spannerpg
// +build !no_spannerpg

package main

//Spanner using PG (postgres) adapter. Using pgx as a driver.
import (
	_ "github.com/jackc/pgx/v5/stdlib"
)
