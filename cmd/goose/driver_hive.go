//go:build !no_hive

package main

import (
	"database/sql"

	"github.com/beltran/gohive/v2"
)

func init() {
	sql.Register("spark", &gohive.Driver{})
}
