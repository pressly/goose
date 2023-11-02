package goose

import (
	"context"
	"database/sql"
	"testing"

	"github.com/pressly/goose/v3/internal/check"
)

func TestNewGoMigration(t *testing.T) {
	t.Run("valid_both_nil", func(t *testing.T) {
		m := NewGoMigration(1, nil, nil)
		// roundtrip
		check.Equal(t, m.Version, int64(1))
		check.Equal(t, m.Type, TypeGo)
		check.Equal(t, m.Registered, true)
		check.Equal(t, m.Next, int64(-1))
		check.Equal(t, m.Previous, int64(-1))
		check.Equal(t, m.Source, "")
		check.Bool(t, m.UpFnNoTxContext == nil, true)
		check.Bool(t, m.DownFnNoTxContext == nil, true)
		check.Bool(t, m.UpFnContext == nil, true)
		check.Bool(t, m.DownFnContext == nil, true)
		check.Bool(t, m.UpFn == nil, true)
		check.Bool(t, m.DownFn == nil, true)
		check.Bool(t, m.UpFnNoTx == nil, true)
		check.Bool(t, m.DownFnNoTx == nil, true)
		check.NotNil(t, m.goUp)
		check.NotNil(t, m.goDown)
		check.Equal(t, m.goUp.Mode, TransactionEnabled)
		check.Equal(t, m.goDown.Mode, TransactionEnabled)
	})
}

func TestTransactionMode(t *testing.T) {
	t.Cleanup(ResetGlobalMigrations)

	runDB := func(context.Context, *sql.DB) error { return nil }
	runTx := func(context.Context, *sql.Tx) error { return nil }

	err := SetGlobalMigrations(
		NewGoMigration(1, &GoFunc{RunTx: runTx, RunDB: runDB}, nil), // cannot specify both
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "up function: must specify exactly one of RunTx or RunDB")
	err = SetGlobalMigrations(
		NewGoMigration(1, nil, &GoFunc{RunTx: runTx, RunDB: runDB}), // cannot specify both
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "down function: must specify exactly one of RunTx or RunDB")
	err = SetGlobalMigrations(
		NewGoMigration(1, &GoFunc{RunTx: runTx, Mode: TransactionDisabled}, nil), // invalid explicit mode tx
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "up function: transaction mode must be enabled or unspecified when RunTx is set")
	err = SetGlobalMigrations(
		NewGoMigration(1, nil, &GoFunc{RunTx: runTx, Mode: TransactionDisabled}), // invalid explicit mode tx
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "down function: transaction mode must be enabled or unspecified when RunTx is set")
	err = SetGlobalMigrations(
		NewGoMigration(1, &GoFunc{RunDB: runDB, Mode: TransactionEnabled}, nil), // invalid explicit mode no-tx
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "up function: transaction mode must be disabled or unspecified when RunDB is set")
	err = SetGlobalMigrations(
		NewGoMigration(1, nil, &GoFunc{RunDB: runDB, Mode: TransactionEnabled}), // invalid explicit mode no-tx
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "down function: transaction mode must be disabled or unspecified when RunDB is set")

	t.Run("default_mode", func(t *testing.T) {
		t.Cleanup(ResetGlobalMigrations)

		m := NewGoMigration(1, nil, nil)
		err = SetGlobalMigrations(m)
		check.NoError(t, err)
		check.Number(t, len(registeredGoMigrations), 1)
		registered := registeredGoMigrations[1]
		check.NotNil(t, registered.goUp)
		check.NotNil(t, registered.goDown)
		check.Equal(t, registered.goUp.Mode, TransactionEnabled)
		check.Equal(t, registered.goDown.Mode, TransactionEnabled)

		migration2 := NewGoMigration(2, nil, nil)
		// reset so we can check the default is set
		migration2.goUp.Mode, migration2.goDown.Mode = 0, 0
		err = SetGlobalMigrations(migration2)
		check.NoError(t, err)
		check.Number(t, len(registeredGoMigrations), 2)
		registered = registeredGoMigrations[2]
		check.NotNil(t, registered.goUp)
		check.NotNil(t, registered.goDown)
		check.Equal(t, registered.goUp.Mode, TransactionEnabled)
		check.Equal(t, registered.goDown.Mode, TransactionEnabled)
	})
	t.Run("unknown_mode", func(t *testing.T) {
		m := NewGoMigration(1, nil, nil)
		m.goUp.Mode, m.goDown.Mode = 3, 3 // reset to default
		err := SetGlobalMigrations(m)
		check.HasError(t, err)
		check.Contains(t, err.Error(), "invalid mode: 3")
	})
}

func TestLegacyFunctions(t *testing.T) {
	t.Cleanup(ResetGlobalMigrations)

	runDB := func(context.Context, *sql.DB) error { return nil }
	runTx := func(context.Context, *sql.Tx) error { return nil }

	assertMigration := func(t *testing.T, m *Migration, version int64) {
		t.Helper()
		check.Equal(t, m.Version, version)
		check.Equal(t, m.Type, TypeGo)
		check.Equal(t, m.Registered, true)
		check.Equal(t, m.Next, int64(-1))
		check.Equal(t, m.Previous, int64(-1))
		check.Equal(t, m.Source, "")
	}

	t.Run("all_tx", func(t *testing.T) {
		err := SetGlobalMigrations(
			NewGoMigration(1, &GoFunc{RunTx: runTx}, &GoFunc{RunTx: runTx}),
		)
		check.NoError(t, err)
		check.Number(t, len(registeredGoMigrations), 1)
		m := registeredGoMigrations[1]
		assertMigration(t, m, 1)
		// Legacy functions.
		check.Bool(t, m.UpFnNoTxContext == nil, true)
		check.Bool(t, m.DownFnNoTxContext == nil, true)
		// Context-aware functions.
		check.Bool(t, m.goUp == nil, false)
		check.Bool(t, m.UpFnContext == nil, false)
		check.Bool(t, m.goDown == nil, false)
		check.Bool(t, m.DownFnContext == nil, false)
		// Always nil
		check.Bool(t, m.UpFn == nil, true)
		check.Bool(t, m.DownFn == nil, true)
		check.Bool(t, m.UpFnNoTx == nil, true)
		check.Bool(t, m.DownFnNoTx == nil, true)
	})
	t.Run("all_db", func(t *testing.T) {
		err := SetGlobalMigrations(
			NewGoMigration(2, &GoFunc{RunDB: runDB}, &GoFunc{RunDB: runDB}),
		)
		check.NoError(t, err)
		check.Number(t, len(registeredGoMigrations), 2)
		m := registeredGoMigrations[2]
		assertMigration(t, m, 2)
		// Legacy functions.
		check.Bool(t, m.UpFnNoTxContext == nil, false)
		check.Bool(t, m.goUp == nil, false)
		check.Bool(t, m.DownFnNoTxContext == nil, false)
		check.Bool(t, m.goDown == nil, false)
		// Context-aware functions.
		check.Bool(t, m.UpFnContext == nil, true)
		check.Bool(t, m.DownFnContext == nil, true)
		// Always nil
		check.Bool(t, m.UpFn == nil, true)
		check.Bool(t, m.DownFn == nil, true)
		check.Bool(t, m.UpFnNoTx == nil, true)
		check.Bool(t, m.DownFnNoTx == nil, true)
	})
}

func TestGlobalRegister(t *testing.T) {
	t.Cleanup(ResetGlobalMigrations)

	runDB := func(context.Context, *sql.DB) error { return nil }
	runTx := func(context.Context, *sql.Tx) error { return nil }

	// Success.
	err := SetGlobalMigrations([]Migration{}...)
	check.NoError(t, err)
	err = SetGlobalMigrations(
		NewGoMigration(1, &GoFunc{RunTx: runTx}, nil),
	)
	check.NoError(t, err)
	// Try to register the same migration again.
	err = SetGlobalMigrations(
		NewGoMigration(1, &GoFunc{RunTx: runTx}, nil),
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "go migration with version 1 already registered")
	err = SetGlobalMigrations(
		Migration{
			Registered:        true,
			Version:           2,
			Source:            "00002_foo.sql",
			Type:              TypeGo,
			UpFnContext:       func(context.Context, *sql.Tx) error { return nil },
			DownFnNoTxContext: func(context.Context, *sql.DB) error { return nil },
		},
	)
	check.NoError(t, err)
	// Reset.
	{
		ResetGlobalMigrations()
	}
	// Failure.
	err = SetGlobalMigrations(
		Migration{},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "invalid go migration: must be registered")
	err = SetGlobalMigrations(
		Migration{Registered: true},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), `invalid go migration: type must be "go"`)
	err = SetGlobalMigrations(
		Migration{Registered: true, Version: 1, Type: TypeSQL},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), `invalid go migration: type must be "go"`)
	err = SetGlobalMigrations(
		Migration{Registered: true, Version: 0, Type: TypeGo},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "invalid go migration: version must be greater than zero")
	err = SetGlobalMigrations(
		Migration{Registered: true, Version: 1, Source: "2_foo.sql", Type: TypeGo},
	)
	check.HasError(t, err)
	check.Contains(
		t,
		err.Error(),
		`invalid go migration: version:1 does not match numeric component in source "2_foo.sql"`,
	)
	// Legacy functions.
	err = SetGlobalMigrations(
		Migration{Registered: true, Version: 1, UpFn: func(tx *sql.Tx) error { return nil }, Type: TypeGo},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "invalid go migration: must not specify UpFn")
	err = SetGlobalMigrations(
		Migration{Registered: true, Version: 1, DownFn: func(tx *sql.Tx) error { return nil }, Type: TypeGo},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "invalid go migration: must not specify DownFn")
	err = SetGlobalMigrations(
		Migration{Registered: true, Version: 1, UpFnNoTx: func(db *sql.DB) error { return nil }, Type: TypeGo},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "invalid go migration: must not specify UpFnNoTx")
	err = SetGlobalMigrations(
		Migration{Registered: true, Version: 1, DownFnNoTx: func(db *sql.DB) error { return nil }, Type: TypeGo},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "invalid go migration: must not specify DownFnNoTx")
	// Context-aware functions.
	err = SetGlobalMigrations(
		Migration{Registered: true, Version: 1, UpFnContext: runTx, UpFnNoTxContext: runDB, Type: TypeGo},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "invalid go migration: must specify exactly one of UpFnContext or UpFnNoTxContext")
	err = SetGlobalMigrations(
		Migration{Registered: true, Version: 1, DownFnContext: runTx, DownFnNoTxContext: runDB, Type: TypeGo},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "invalid go migration: must specify exactly one of DownFnContext or DownFnNoTxContext")
	// Source and version mismatch.
	err = SetGlobalMigrations(
		Migration{Registered: true, Version: 1, Source: "invalid_numeric.sql", Type: TypeGo},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), `invalid go migration: failed to parse version from migration file: invalid_numeric.sql`)
}
