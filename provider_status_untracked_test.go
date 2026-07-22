package goose_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestProviderStatusIncludesDBOnlyApplied(t *testing.T) {
	ctx := context.Background()
	dbName := filepath.Join(t.TempDir(), "status_untracked.db")
	db, err := sql.Open("sqlite", dbName)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	// Provider with only version 1 source.
	fsys := fstest.MapFS{
		"00001_a.sql": {Data: []byte(`-- +goose Up
CREATE TABLE a (id INTEGER);
-- +goose Down
DROP TABLE a;
`)},
	}
	p, err := goose.NewProvider(goose.DialectSQLite3, db, fsys)
	require.NoError(t, err)
	_, err = p.Up(ctx)
	require.NoError(t, err)

	// Insert an applied version that has no local source (simulates production-only migration).
	_, err = db.Exec(`INSERT INTO goose_db_version (version_id, is_applied) VALUES (99, 1)`)
	require.NoError(t, err)

	status, err := p.Status(ctx)
	require.NoError(t, err)

	var found *goose.MigrationStatus
	for _, s := range status {
		if s.Source != nil && s.Source.Version == 99 {
			found = s
			break
		}
	}
	require.NotNil(t, found, "expected untracked version 99 in status")
	require.Equal(t, goose.StateUntracked, found.State)
	require.True(t, found.Source.Path == "")
}
