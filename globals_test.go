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
		check.Bool(t, m.goUp != nil, true)
		check.Bool(t, m.goDown != nil, true)
		check.Equal(t, m.goUp.Mode, TransactionEnabled)
		check.Equal(t, m.goDown.Mode, TransactionEnabled)
	})
	t.Run("all_set", func(t *testing.T) {
		// This will eventually be an error when registering migrations.
		m := NewGoMigration(
			1,
			&GoFunc{RunTx: func(context.Context, *sql.Tx) error { return nil }, RunDB: func(context.Context, *sql.DB) error { return nil }},
			&GoFunc{RunTx: func(context.Context, *sql.Tx) error { return nil }, RunDB: func(context.Context, *sql.DB) error { return nil }},
		)
		// check only functions
		check.Bool(t, m.UpFn != nil, true)
		check.Bool(t, m.UpFnContext != nil, true)
		check.Bool(t, m.UpFnNoTx != nil, true)
		check.Bool(t, m.UpFnNoTxContext != nil, true)
		check.Bool(t, m.DownFn != nil, true)
		check.Bool(t, m.DownFnContext != nil, true)
		check.Bool(t, m.DownFnNoTx != nil, true)
		check.Bool(t, m.DownFnNoTxContext != nil, true)
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
		check.Bool(t, registered.goUp != nil, true)
		check.Bool(t, registered.goDown != nil, true)
		check.Equal(t, registered.goUp.Mode, TransactionEnabled)
		check.Equal(t, registered.goDown.Mode, TransactionEnabled)

		migration2 := NewGoMigration(2, nil, nil)
		// reset so we can check the default is set
		migration2.goUp.Mode, migration2.goDown.Mode = 0, 0
		err = SetGlobalMigrations(migration2)
		check.HasError(t, err)
		check.Contains(t, err.Error(), "invalid go migration: up function: invalid mode: 0")

		migration3 := NewGoMigration(3, nil, nil)
		// reset so we can check the default is set
		migration3.goDown.Mode = 0
		err = SetGlobalMigrations(migration3)
		check.HasError(t, err)
		check.Contains(t, err.Error(), "invalid go migration: down function: invalid mode: 0")
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
		t.Cleanup(ResetGlobalMigrations)
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
		check.Bool(t, m.UpFn == nil, false)
		check.Bool(t, m.DownFn == nil, false)
		check.Bool(t, m.UpFnNoTx == nil, true)
		check.Bool(t, m.DownFnNoTx == nil, true)
	})
	t.Run("all_db", func(t *testing.T) {
		t.Cleanup(ResetGlobalMigrations)
		err := SetGlobalMigrations(
			NewGoMigration(2, &GoFunc{RunDB: runDB}, &GoFunc{RunDB: runDB}),
		)
		check.NoError(t, err)
		check.Number(t, len(registeredGoMigrations), 1)
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
		check.Bool(t, m.UpFnNoTx == nil, false)
		check.Bool(t, m.DownFnNoTx == nil, false)
	})
}

func TestGlobalRegister(t *testing.T) {
	t.Cleanup(ResetGlobalMigrations)

	// runDB := func(context.Context, *sql.DB) error { return nil }
	runTx := func(context.Context, *sql.Tx) error { return nil }

	// Success.
	err := SetGlobalMigrations([]*Migration{}...)
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
	err = SetGlobalMigrations(&Migration{Registered: true, Version: 2, Type: TypeGo})
	check.HasError(t, err)
	check.Contains(t, err.Error(), "must use NewGoMigration to construct migrations")
}

func TestCheckMigration(t *testing.T) {
	// Success.
	err := checkGoMigration(NewGoMigration(1, nil, nil))
	check.NoError(t, err)
	// Failures.
	err = checkGoMigration(&Migration{})
	check.HasError(t, err)
	check.Contains(t, err.Error(), "must use NewGoMigration to construct migrations")
	err = checkGoMigration(&Migration{construct: true})
	check.HasError(t, err)
	check.Contains(t, err.Error(), "must be registered")
	err = checkGoMigration(&Migration{construct: true, Registered: true})
	check.HasError(t, err)
	check.Contains(t, err.Error(), `type must be "go"`)
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo})
	check.HasError(t, err)
	check.Contains(t, err.Error(), "version must be greater than zero")
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 1, goUp: &GoFunc{}, goDown: &GoFunc{}})
	check.HasError(t, err)
	check.Contains(t, err.Error(), "up function: invalid mode: 0")
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 1, goUp: &GoFunc{Mode: TransactionEnabled}, goDown: &GoFunc{}})
	check.HasError(t, err)
	check.Contains(t, err.Error(), "down function: invalid mode: 0")
	// Success.
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 1, goUp: &GoFunc{Mode: TransactionEnabled}, goDown: &GoFunc{Mode: TransactionEnabled}})
	check.NoError(t, err)
	// Failures.
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 1, Source: "foo"})
	check.HasError(t, err)
	check.Contains(t, err.Error(), `source must have .go extension: "foo"`)
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 1, Source: "foo.go"})
	check.HasError(t, err)
	check.Contains(t, err.Error(), `no filename separator '_' found`)
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 2, Source: "00001_foo.sql"})
	check.HasError(t, err)
	check.Contains(t, err.Error(), `source must have .go extension: "00001_foo.sql"`)
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 2, Source: "00001_foo.go"})
	check.HasError(t, err)
	check.Contains(t, err.Error(), `version:2 does not match numeric component in source "00001_foo.go"`)
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 1,
		UpFnContext:     func(context.Context, *sql.Tx) error { return nil },
		UpFnNoTxContext: func(context.Context, *sql.DB) error { return nil },
		goUp:            &GoFunc{Mode: TransactionEnabled},
		goDown:          &GoFunc{Mode: TransactionEnabled},
	})
	check.HasError(t, err)
	check.Contains(t, err.Error(), "must specify exactly one of UpFnContext or UpFnNoTxContext")
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 1,
		DownFnContext:     func(context.Context, *sql.Tx) error { return nil },
		DownFnNoTxContext: func(context.Context, *sql.DB) error { return nil },
		goUp:              &GoFunc{Mode: TransactionEnabled},
		goDown:            &GoFunc{Mode: TransactionEnabled},
	})
	check.HasError(t, err)
	check.Contains(t, err.Error(), "must specify exactly one of DownFnContext or DownFnNoTxContext")
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 1,
		UpFn:     func(*sql.Tx) error { return nil },
		UpFnNoTx: func(*sql.DB) error { return nil },
		goUp:     &GoFunc{Mode: TransactionEnabled},
		goDown:   &GoFunc{Mode: TransactionEnabled},
	})
	check.HasError(t, err)
	check.Contains(t, err.Error(), "must specify exactly one of UpFn or UpFnNoTx")
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 1,
		DownFn:     func(*sql.Tx) error { return nil },
		DownFnNoTx: func(*sql.DB) error { return nil },
		goUp:       &GoFunc{Mode: TransactionEnabled},
		goDown:     &GoFunc{Mode: TransactionEnabled},
	})
	check.HasError(t, err)
	check.Contains(t, err.Error(), "must specify exactly one of DownFn or DownFnNoTx")
}
