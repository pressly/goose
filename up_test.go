package goose

import (
	"testing"

	"github.com/matryer/is"
)

func TestFindMissingMigrations(t *testing.T) {
	is := is.New(t)
	known := Migrations{
		{Version: 1},
		{Version: 3},
		{Version: 4},
		{Version: 5},
		{Version: 7},
	}
	new := Migrations{
		{Version: 1},
		{Version: 2}, // missing migration
		{Version: 3},
		{Version: 4},
		{Version: 5},
		{Version: 6}, // missing migration
		{Version: 7}, // <-- database max version_id
		{Version: 8}, // new migration
	}
	got := findMissingMigrations(known, new)
	is.Equal(len(got), int(2))
	is.Equal(got[0].Version, int64(2)) // Expecting first missing migration
	is.Equal(got[1].Version, int64(6)) // Expecting second missing migration
}
