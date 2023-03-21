package goose

import (
	"testing"

	"github.com/pressly/goose/v4/internal/check"
	"github.com/pressly/goose/v4/internal/dialectadapter"
	"github.com/pressly/goose/v4/internal/migration"
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
		{Version: 7}, // <-- database max version_id
	}
	fsMigrations := []*migration.Migration{
		newMigration(1),
		newMigration(2), // missing migration
		newMigration(3),
		newMigration(4),
		newMigration(5),
		newMigration(6), // missing migration
		newMigration(7), // ----- database max version_id -----
		newMigration(8), // new migration
	}
	got := findMissingMigrations(dbMigrations, fsMigrations, 7)
	check.Number(t, len(got), 2)
	check.Number(t, got[0].versionID, 2)
	check.Number(t, got[1].versionID, 6)

	// Sanity check.
	check.Number(t, len(findMissingMigrations(nil, nil, 0)), 0)
	check.Number(t, len(findMissingMigrations(dbMigrations, nil, 0)), 0)
	check.Number(t, len(findMissingMigrations(nil, fsMigrations, 0)), 0)
}

func newMigration(version int64) *migration.Migration {
	return &migration.Migration{
		Version: version,
	}
}
