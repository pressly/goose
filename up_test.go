package goose

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaxDBVersion(t *testing.T) {
	t.Parallel()

	// An empty list must not panic and is treated as version 0. This can happen when the
	// version table row is not visible to the current connection.
	require.EqualValues(t, 0, maxDBVersion(nil))
	require.EqualValues(t, 0, maxDBVersion(Migrations{}))
	require.EqualValues(t, 2, maxDBVersion(Migrations{{Version: 1}, {Version: 2}}))
}

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
