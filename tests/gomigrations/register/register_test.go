package register_test

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	_ "github.com/pressly/goose/v3/tests/gomigrations/register/testdata"
	"github.com/stretchr/testify/require"
)

func TestAddFunctions(t *testing.T) {
	goMigrations, err := goose.CollectMigrations("testdata", 0, math.MaxInt64)
	require.NoError(t, err)
	require.Equal(t, len(goMigrations), 4)

	checkMigration(t, goMigrations[0], &goose.Migration{
		Version:    1,
		Next:       2,
		Previous:   -1,
		Source:     "001_addmigration.go",
		Registered: true,
		UseTx:      true,
	})
	checkMigration(t, goMigrations[1], &goose.Migration{
		Version:    2,
		Next:       3,
		Previous:   1,
		Source:     "002_addmigrationnotx.go",
		Registered: true,
		UseTx:      false,
	})
	checkMigration(t, goMigrations[2], &goose.Migration{
		Version:    3,
		Next:       4,
		Previous:   2,
		Source:     "003_addmigrationcontext.go",
		Registered: true,
		UseTx:      true,
	})
	checkMigration(t, goMigrations[3], &goose.Migration{
		Version:    4,
		Next:       -1,
		Previous:   3,
		Source:     "004_addmigrationnotxcontext.go",
		Registered: true,
		UseTx:      false,
	})
}

func checkMigration(t *testing.T, got *goose.Migration, want *goose.Migration) {
	t.Helper()
	require.Equal(t, got.Version, want.Version)
	require.Equal(t, got.Next, want.Next)
	require.Equal(t, got.Previous, want.Previous)
	require.Equal(t, filepath.Base(got.Source), want.Source)
	require.Equal(t, got.Registered, want.Registered)
	require.Equal(t, got.UseTx, want.UseTx)
	checkFunctions(t, got)
}

func checkFunctions(t *testing.T, m *goose.Migration) {
	t.Helper()
	switch filepath.Base(m.Source) {
	case "001_addmigration.go":
		// With transaction
		require.False(t, m.UpFn == nil)
		require.False(t, m.DownFn == nil)
		require.False(t, m.UpFnContext == nil)
		require.False(t, m.DownFnContext == nil)
		// No transaction
		require.True(t, m.UpFnNoTx == nil)
		require.True(t, m.DownFnNoTx == nil)
		require.True(t, m.UpFnNoTxContext == nil)
		require.True(t, m.DownFnNoTxContext == nil)
	case "002_addmigrationnotx.go":
		// With transaction
		require.True(t, m.UpFn == nil)
		require.True(t, m.DownFn == nil)
		require.True(t, m.UpFnContext == nil)
		require.True(t, m.DownFnContext == nil)
		// No transaction
		require.False(t, m.UpFnNoTx == nil)
		require.False(t, m.DownFnNoTx == nil)
		require.False(t, m.UpFnNoTxContext == nil)
		require.False(t, m.DownFnNoTxContext == nil)
	case "003_addmigrationcontext.go":
		// With transaction
		require.False(t, m.UpFn == nil)
		require.False(t, m.DownFn == nil)
		require.False(t, m.UpFnContext == nil)
		require.False(t, m.DownFnContext == nil)
		// No transaction
		require.True(t, m.UpFnNoTx == nil)
		require.True(t, m.DownFnNoTx == nil)
		require.True(t, m.UpFnNoTxContext == nil)
		require.True(t, m.DownFnNoTxContext == nil)
	case "004_addmigrationnotxcontext.go":
		// With transaction
		require.True(t, m.UpFn == nil)
		require.True(t, m.DownFn == nil)
		require.True(t, m.UpFnContext == nil)
		require.True(t, m.DownFnContext == nil)
		// No transaction
		require.False(t, m.UpFnNoTx == nil)
		require.False(t, m.DownFnNoTx == nil)
		require.False(t, m.UpFnNoTxContext == nil)
		require.False(t, m.DownFnNoTxContext == nil)
	default:
		t.Fatalf("unexpected migration: %s", filepath.Base(m.Source))
	}
}
