// +build dev

package postgres

//go:generate go run -tags=dev ../assets_generate.go

import "net/http"

// Migrations contains project assets.
var Migrations http.FileSystem = http.Dir("migrations")