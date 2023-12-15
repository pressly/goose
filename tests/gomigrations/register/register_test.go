package register_test

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
	_ "github.com/pressly/goose/v3/tests/gomigrations/register/testdata"
)

func TestAddFunctions(t *testing.T) {
	goMigrations, err := goose.CollectMigrations("testdata", 0, math.MaxInt64)
	check.NoError(t, err)
	check.Number(t, len(goMigrations), 4)

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
	check.Equal(t, got.Version, want.Version)
	check.Equal(t, got.Next, want.Next)
	check.Equal(t, got.Previous, want.Previous)
	check.Equal(t, filepath.Base(got.Source), want.Source)
	check.Equal(t, got.Registered, want.Registered)
	check.Bool(t, got.UseTx, want.UseTx)
	checkFunctions(t, got)
}

func checkFunctions(t *testing.T, m *goose.Migration) {
	t.Helper()
	switch filepath.Base(m.Source) {
	case "001_addmigration.go":
		// With transaction
		check.Bool(t, m.UpFn == nil, false)
		check.Bool(t, m.DownFn == nil, false)
		check.Bool(t, m.UpFnContext == nil, false)
		check.Bool(t, m.DownFnContext == nil, false)
		// No transaction
		check.Bool(t, m.UpFnNoTx == nil, true)
		check.Bool(t, m.DownFnNoTx == nil, true)
		check.Bool(t, m.UpFnNoTxContext == nil, true)
		check.Bool(t, m.DownFnNoTxContext == nil, true)
	case "002_addmigrationnotx.go":
		// With transaction
		check.Bool(t, m.UpFn == nil, true)
		check.Bool(t, m.DownFn == nil, true)
		check.Bool(t, m.UpFnContext == nil, true)
		check.Bool(t, m.DownFnContext == nil, true)
		// No transaction
		check.Bool(t, m.UpFnNoTx == nil, false)
		check.Bool(t, m.DownFnNoTx == nil, false)
		check.Bool(t, m.UpFnNoTxContext == nil, false)
		check.Bool(t, m.DownFnNoTxContext == nil, false)
	case "003_addmigrationcontext.go":
		// With transaction
		check.Bool(t, m.UpFn == nil, false)
		check.Bool(t, m.DownFn == nil, false)
		check.Bool(t, m.UpFnContext == nil, false)
		check.Bool(t, m.DownFnContext == nil, false)
		// No transaction
		check.Bool(t, m.UpFnNoTx == nil, true)
		check.Bool(t, m.DownFnNoTx == nil, true)
		check.Bool(t, m.UpFnNoTxContext == nil, true)
		check.Bool(t, m.DownFnNoTxContext == nil, true)
	case "004_addmigrationnotxcontext.go":
		// With transaction
		check.Bool(t, m.UpFn == nil, true)
		check.Bool(t, m.DownFn == nil, true)
		check.Bool(t, m.UpFnContext == nil, true)
		check.Bool(t, m.DownFnContext == nil, true)
		// No transaction
		check.Bool(t, m.UpFnNoTx == nil, false)
		check.Bool(t, m.DownFnNoTx == nil, false)
		check.Bool(t, m.UpFnNoTxContext == nil, false)
		check.Bool(t, m.DownFnNoTxContext == nil, false)
	default:
		t.Fatalf("unexpected migration: %s", filepath.Base(m.Source))
	}
}
