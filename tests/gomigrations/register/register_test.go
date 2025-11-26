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
	require.Len(t, goMigrations, 4)

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
	require.Equal(t, want.Version, got.Version)
	require.Equal(t, want.Next, got.Next)
	require.Equal(t, want.Previous, got.Previous)
	require.Equal(t, want.Source, filepath.Base(got.Source))
	require.Equal(t, want.Registered, got.Registered)
	require.Equal(t, want.UseTx, got.UseTx)
	checkFunctions(t, got)
}

func checkFunctions(t *testing.T, m *goose.Migration) {
	t.Helper()
	switch filepath.Base(m.Source) {
	case "001_addmigration.go":
		// With transaction
		require.NotNil(t, m.UpFn)
		require.NotNil(t, m.DownFn)
		require.NotNil(t, m.UpFnContext)
		require.NotNil(t, m.DownFnContext)
		// No transaction
		require.Nil(t, m.UpFnNoTx)
		require.Nil(t, m.DownFnNoTx)
		require.Nil(t, m.UpFnNoTxContext)
		require.Nil(t, m.DownFnNoTxContext)
	case "002_addmigrationnotx.go":
		// With transaction
		require.Nil(t, m.UpFn)
		require.Nil(t, m.DownFn)
		require.Nil(t, m.UpFnContext)
		require.Nil(t, m.DownFnContext)
		// No transaction
		require.NotNil(t, m.UpFnNoTx)
		require.NotNil(t, m.DownFnNoTx)
		require.NotNil(t, m.UpFnNoTxContext)
		require.NotNil(t, m.DownFnNoTxContext)
	case "003_addmigrationcontext.go":
		// With transaction
		require.NotNil(t, m.UpFn)
		require.NotNil(t, m.DownFn)
		require.NotNil(t, m.UpFnContext)
		require.NotNil(t, m.DownFnContext)
		// No transaction
		require.Nil(t, m.UpFnNoTx)
		require.Nil(t, m.DownFnNoTx)
		require.Nil(t, m.UpFnNoTxContext)
		require.Nil(t, m.DownFnNoTxContext)
	case "004_addmigrationnotxcontext.go":
		// With transaction
		require.Nil(t, m.UpFn)
		require.Nil(t, m.DownFn)
		require.Nil(t, m.UpFnContext)
		require.Nil(t, m.DownFnContext)
		// No transaction
		require.NotNil(t, m.UpFnNoTx)
		require.NotNil(t, m.DownFnNoTx)
		require.NotNil(t, m.UpFnNoTxContext)
		require.NotNil(t, m.DownFnNoTxContext)
	default:
		t.Fatalf("unexpected migration: %s", filepath.Base(m.Source))
	}
}
