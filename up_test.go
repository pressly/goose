package goose

import (
	"testing"

	"github.com/pressly/goose/v4/internal/check"
	"github.com/pressly/goose/v4/internal/dialectadapter"
)

func TestFindMissingMigrations(t *testing.T) {
	t.Parallel()

	// Test case: database has migrations 1, 3, 4, 5, 7
	// Missing migrations: 2, 6
	// Filesystem has migrations 1, 2, 3, 4, 5, 6, 7, 8

	dbMigrations := []*dialectadapter.ListMigrationsResult{
		{Version: 1},
		{Version: 3},
		{Version: 4},
		{Version: 5},
		{Version: 7},
	}
	fsMigrations := []*migration{
		{version: 1},
		{version: 2}, // missing migration
		{version: 3},
		{version: 4},
		{version: 5},
		{version: 6}, // missing migration
		{version: 7}, // ----- database max version_id -----
		{version: 8}, // new migration
	}
	got := findMissingMigrations(dbMigrations, fsMigrations)
	check.Number(t, len(got), 2)
	check.Number(t, got[0], 2)
	check.Number(t, got[1], 6)

	// Sanity check.
	check.Number(t, len(findMissingMigrations(nil, nil)), 0)
	check.Number(t, len(findMissingMigrations(dbMigrations, nil)), 0)
	check.Number(t, len(findMissingMigrations(nil, fsMigrations)), 0)
}
