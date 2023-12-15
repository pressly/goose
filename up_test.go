package goose

import (
	"testing"
)

func TestFindMissingMigrations(t *testing.T) {
	known := Migrations{
		{Version: 1},
		{Version: 3},
		{Version: 4},
		{Version: 5},
		{Version: 7}, // <-- database max version_id
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
	got := findMissingMigrations(known, new, 7)
	if len(got) != 2 {
		t.Fatalf("invalid migration count: got:%d want:%d", len(got), 2)
	}
	if got[0].Version != 2 {
		t.Errorf("expecting first migration: got:%d want:%d", got[0].Version, 2)
	}
	if got[1].Version != 6 {
		t.Errorf("expecting second migration: got:%d want:%d", got[0].Version, 6)
	}
}
