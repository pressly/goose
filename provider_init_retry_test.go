package goose_test

import (
	"context"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestProviderRetryAfterVersionTableFailure(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := newDB(t)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	// Another connection holds the database lock during the first migration attempt.
	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	locked := false
	t.Cleanup(func() {
		if locked {
			_, err := conn.ExecContext(ctx, "ROLLBACK")
			require.NoError(t, err)
		}
		require.NoError(t, conn.Close())
	})
	_, err = conn.ExecContext(ctx, "BEGIN EXCLUSIVE")
	require.NoError(t, err)
	locked = true

	p, err := goose.NewProvider(goose.DialectSQLite3, db, newFsys())
	require.NoError(t, err)
	_, err = p.Up(ctx)
	require.ErrorContains(t, err, "database is locked")

	_, err = conn.ExecContext(ctx, "ROLLBACK")
	require.NoError(t, err)
	locked = false

	// Retrying the same provider after the lock is released must initialize the version
	// table and apply the migrations, without requiring a new provider or process restart.
	results, err := p.Up(ctx)
	require.NoError(t, err)
	require.Len(t, results, len(newFsys()))
	current, target, err := p.GetVersions(ctx)
	require.NoError(t, err)
	require.EqualValues(t, len(newFsys()), current)
	require.Equal(t, current, target)

	results, err = p.Up(ctx)
	require.NoError(t, err)
	require.Empty(t, results)
}
