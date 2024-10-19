package goose

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewGoMigration(t *testing.T) {
	t.Run("valid_both_nil", func(t *testing.T) {
		m := NewGoMigration(1, nil, nil)
		// roundtrip
		require.EqualValues(t, 1, m.Version)
		require.Equal(t, TypeGo, m.Type)
		require.True(t, m.Registered)
		require.EqualValues(t, -1, m.Next)
		require.EqualValues(t, -1, m.Previous)
		require.Equal(t, "", m.Source)
		require.Nil(t, m.UpFnNoTxContext)
		require.Nil(t, m.DownFnNoTxContext)
		require.Nil(t, m.UpFnContext)
		require.Nil(t, m.DownFnContext)
		require.Nil(t, m.UpFn)
		require.Nil(t, m.DownFn)
		require.Nil(t, m.UpFnNoTx)
		require.Nil(t, m.DownFnNoTx)
		require.NotNil(t, m.goUp)
		require.NotNil(t, m.goDown)
		require.Equal(t, TransactionEnabled, m.goUp.Mode)
		require.Equal(t, TransactionEnabled, m.goDown.Mode)
	})
	t.Run("all_set", func(t *testing.T) {
		// This will eventually be an error when registering migrations.
		m := NewGoMigration(
			1,
			&GoFunc{RunTx: func(context.Context, *sql.Tx) error { return nil }, RunDB: func(context.Context, *sql.DB) error { return nil }},
			&GoFunc{RunTx: func(context.Context, *sql.Tx) error { return nil }, RunDB: func(context.Context, *sql.DB) error { return nil }},
		)
		// check only functions
		require.NotNil(t, m.UpFn)
		require.NotNil(t, m.UpFnContext)
		require.NotNil(t, m.UpFnNoTx)
		require.NotNil(t, m.UpFnNoTxContext)
		require.NotNil(t, m.DownFn)
		require.NotNil(t, m.DownFnContext)
		require.NotNil(t, m.DownFnNoTx)
		require.NotNil(t, m.DownFnNoTxContext)
	})
}

func TestTransactionMode(t *testing.T) {
	t.Cleanup(ResetGlobalMigrations)

	runDB := func(context.Context, *sql.DB) error { return nil }
	runTx := func(context.Context, *sql.Tx) error { return nil }

	err := SetGlobalMigrations(
		NewGoMigration(1, &GoFunc{RunTx: runTx, RunDB: runDB}, nil), // cannot specify both
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "up function: must specify exactly one of RunTx or RunDB")
	err = SetGlobalMigrations(
		NewGoMigration(1, nil, &GoFunc{RunTx: runTx, RunDB: runDB}), // cannot specify both
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "down function: must specify exactly one of RunTx or RunDB")
	err = SetGlobalMigrations(
		NewGoMigration(1, &GoFunc{RunTx: runTx, Mode: TransactionDisabled}, nil), // invalid explicit mode tx
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "up function: transaction mode must be enabled or unspecified when RunTx is set")
	err = SetGlobalMigrations(
		NewGoMigration(1, nil, &GoFunc{RunTx: runTx, Mode: TransactionDisabled}), // invalid explicit mode tx
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "down function: transaction mode must be enabled or unspecified when RunTx is set")
	err = SetGlobalMigrations(
		NewGoMigration(1, &GoFunc{RunDB: runDB, Mode: TransactionEnabled}, nil), // invalid explicit mode no-tx
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "up function: transaction mode must be disabled or unspecified when RunDB is set")
	err = SetGlobalMigrations(
		NewGoMigration(1, nil, &GoFunc{RunDB: runDB, Mode: TransactionEnabled}), // invalid explicit mode no-tx
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "down function: transaction mode must be disabled or unspecified when RunDB is set")

	t.Run("default_mode", func(t *testing.T) {
		t.Cleanup(ResetGlobalMigrations)

		m := NewGoMigration(1, nil, nil)
		err = SetGlobalMigrations(m)
		require.NoError(t, err)
		require.Len(t, registeredGoMigrations, 1)
		registered := registeredGoMigrations[1]
		require.NotNil(t, registered.goUp)
		require.NotNil(t, registered.goDown)
		require.Equal(t, TransactionEnabled, registered.goUp.Mode)
		require.Equal(t, TransactionEnabled, registered.goDown.Mode)

		migration2 := NewGoMigration(2, nil, nil)
		// reset so we can check the default is set
		migration2.goUp.Mode, migration2.goDown.Mode = 0, 0
		err = SetGlobalMigrations(migration2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid go migration: up function: invalid mode: 0")

		migration3 := NewGoMigration(3, nil, nil)
		// reset so we can check the default is set
		migration3.goDown.Mode = 0
		err = SetGlobalMigrations(migration3)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid go migration: down function: invalid mode: 0")
	})
	t.Run("unknown_mode", func(t *testing.T) {
		m := NewGoMigration(1, nil, nil)
		m.goUp.Mode, m.goDown.Mode = 3, 3 // reset to default
		err := SetGlobalMigrations(m)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid mode: 3")
	})
}

func TestLegacyFunctions(t *testing.T) {
	t.Cleanup(ResetGlobalMigrations)

	runDB := func(context.Context, *sql.DB) error { return nil }
	runTx := func(context.Context, *sql.Tx) error { return nil }

	assertMigration := func(t *testing.T, m *Migration, version int64) {
		t.Helper()
		require.Equal(t, version, m.Version)
		require.Equal(t, TypeGo, m.Type)
		require.True(t, m.Registered)
		require.EqualValues(t, -1, m.Next)
		require.EqualValues(t, -1, m.Previous)
		require.Equal(t, "", m.Source)
	}

	t.Run("all_tx", func(t *testing.T) {
		t.Cleanup(ResetGlobalMigrations)
		err := SetGlobalMigrations(
			NewGoMigration(1, &GoFunc{RunTx: runTx}, &GoFunc{RunTx: runTx}),
		)
		require.NoError(t, err)
		require.Len(t, registeredGoMigrations, 1)
		m := registeredGoMigrations[1]
		assertMigration(t, m, 1)
		// Legacy functions.
		require.Nil(t, m.UpFnNoTxContext)
		require.Nil(t, m.DownFnNoTxContext)
		// Context-aware functions.
		require.NotNil(t, m.goUp)
		require.NotNil(t, m.UpFnContext)
		require.NotNil(t, m.goDown)
		require.NotNil(t, m.DownFnContext)
		// Always nil
		require.NotNil(t, m.UpFn)
		require.NotNil(t, m.DownFn)
		require.Nil(t, m.UpFnNoTx)
		require.Nil(t, m.DownFnNoTx)
	})
	t.Run("all_db", func(t *testing.T) {
		t.Cleanup(ResetGlobalMigrations)
		err := SetGlobalMigrations(
			NewGoMigration(2, &GoFunc{RunDB: runDB}, &GoFunc{RunDB: runDB}),
		)
		require.NoError(t, err)
		require.Len(t, registeredGoMigrations, 1)
		m := registeredGoMigrations[2]
		assertMigration(t, m, 2)
		// Legacy functions.
		require.NotNil(t, m.UpFnNoTxContext)
		require.NotNil(t, m.goUp)
		require.NotNil(t, m.DownFnNoTxContext)
		require.NotNil(t, m.goDown)
		// Context-aware functions.
		require.Nil(t, m.UpFnContext)
		require.Nil(t, m.DownFnContext)
		// Always nil
		require.Nil(t, m.UpFn)
		require.Nil(t, m.DownFn)
		require.NotNil(t, m.UpFnNoTx)
		require.NotNil(t, m.DownFnNoTx)
	})
}

func TestGlobalRegister(t *testing.T) {
	t.Cleanup(ResetGlobalMigrations)

	// runDB := func(context.Context, *sql.DB) error { return nil }
	runTx := func(context.Context, *sql.Tx) error { return nil }

	// Success.
	err := SetGlobalMigrations([]*Migration{}...)
	require.NoError(t, err)
	err = SetGlobalMigrations(
		NewGoMigration(1, &GoFunc{RunTx: runTx}, nil),
	)
	require.NoError(t, err)
	// Try to register the same migration again.
	err = SetGlobalMigrations(
		NewGoMigration(1, &GoFunc{RunTx: runTx}, nil),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "go migration with version 1 already registered")
	err = SetGlobalMigrations(&Migration{Registered: true, Version: 2, Type: TypeGo})
	require.Error(t, err)
	require.Contains(t, err.Error(), "must use NewGoMigration to construct migrations")
}

func TestCheckMigration(t *testing.T) {
	// Success.
	err := checkGoMigration(NewGoMigration(1, nil, nil))
	require.NoError(t, err)
	// Failures.
	err = checkGoMigration(&Migration{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "must use NewGoMigration to construct migrations")
	err = checkGoMigration(&Migration{construct: true})
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be registered")
	err = checkGoMigration(&Migration{construct: true, Registered: true})
	require.Error(t, err)
	require.Contains(t, err.Error(), `type must be "go"`)
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo})
	require.Error(t, err)
	require.Contains(t, err.Error(), "version must be greater than zero")
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 1, goUp: &GoFunc{}, goDown: &GoFunc{}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "up function: invalid mode: 0")
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 1, goUp: &GoFunc{Mode: TransactionEnabled}, goDown: &GoFunc{}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "down function: invalid mode: 0")
	// Success.
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 1, goUp: &GoFunc{Mode: TransactionEnabled}, goDown: &GoFunc{Mode: TransactionEnabled}})
	require.NoError(t, err)
	// Failures.
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 1, Source: "foo"})
	require.Error(t, err)
	require.Contains(t, err.Error(), `source must have .go extension: "foo"`)
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 1, Source: "foo.go"})
	require.Error(t, err)
	require.Contains(t, err.Error(), `no filename separator '_' found`)
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 2, Source: "00001_foo.sql"})
	require.Error(t, err)
	require.Contains(t, err.Error(), `source must have .go extension: "00001_foo.sql"`)
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 2, Source: "00001_foo.go"})
	require.Error(t, err)
	require.Contains(t, err.Error(), `version:2 does not match numeric component in source "00001_foo.go"`)
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 1,
		UpFnContext:     func(context.Context, *sql.Tx) error { return nil },
		UpFnNoTxContext: func(context.Context, *sql.DB) error { return nil },
		goUp:            &GoFunc{Mode: TransactionEnabled},
		goDown:          &GoFunc{Mode: TransactionEnabled},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "must specify exactly one of UpFnContext or UpFnNoTxContext")
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 1,
		DownFnContext:     func(context.Context, *sql.Tx) error { return nil },
		DownFnNoTxContext: func(context.Context, *sql.DB) error { return nil },
		goUp:              &GoFunc{Mode: TransactionEnabled},
		goDown:            &GoFunc{Mode: TransactionEnabled},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "must specify exactly one of DownFnContext or DownFnNoTxContext")
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 1,
		UpFn:     func(*sql.Tx) error { return nil },
		UpFnNoTx: func(*sql.DB) error { return nil },
		goUp:     &GoFunc{Mode: TransactionEnabled},
		goDown:   &GoFunc{Mode: TransactionEnabled},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "must specify exactly one of UpFn or UpFnNoTx")
	err = checkGoMigration(&Migration{construct: true, Registered: true, Type: TypeGo, Version: 1,
		DownFn:     func(*sql.Tx) error { return nil },
		DownFnNoTx: func(*sql.DB) error { return nil },
		goUp:       &GoFunc{Mode: TransactionEnabled},
		goDown:     &GoFunc{Mode: TransactionEnabled},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "must specify exactly one of DownFn or DownFnNoTx")
}
