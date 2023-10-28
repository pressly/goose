package goose_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
)

func TestGlobalRegister(t *testing.T) {
	// Avoid polluting other tests and do not run in parallel.
	t.Cleanup(func() {
		goose.ResetGlobalMigrations()
	})
	fnNoTx := func(context.Context, *sql.DB) error { return nil }
	fn := func(context.Context, *sql.Tx) error { return nil }

	// Success.
	err := goose.SetGlobalMigrations(
		[]goose.Migration{}...,
	)
	check.NoError(t, err)
	err = goose.SetGlobalMigrations(
		goose.Migration{Registered: true, Version: 1, Type: goose.TypeGo, UpFnContext: fn},
	)
	check.NoError(t, err)
	err = goose.SetGlobalMigrations(
		goose.Migration{Registered: true, Version: 1, Type: goose.TypeGo},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "go migration with version 1 already registered")
	err = goose.SetGlobalMigrations(
		goose.Migration{
			Registered:        true,
			Version:           2,
			Source:            "00002_foo.sql",
			Type:              goose.TypeGo,
			UpFnContext:       func(context.Context, *sql.Tx) error { return nil },
			DownFnNoTxContext: func(context.Context, *sql.DB) error { return nil },
		},
	)
	check.NoError(t, err)
	// Reset.
	{
		goose.ResetGlobalMigrations()
	}
	// Failure.
	err = goose.SetGlobalMigrations(
		goose.Migration{},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "invalid go migration: must be registered")
	err = goose.SetGlobalMigrations(
		goose.Migration{Registered: true},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), `invalid go migration: type must be "go"`)
	err = goose.SetGlobalMigrations(
		goose.Migration{Registered: true, Version: 1, Type: goose.TypeSQL},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), `invalid go migration: type must be "go"`)
	err = goose.SetGlobalMigrations(
		goose.Migration{Registered: true, Version: 0, Type: goose.TypeGo},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "invalid go migration: version must be greater than zero")
	err = goose.SetGlobalMigrations(
		goose.Migration{Registered: true, Version: 1, Source: "2_foo.sql", Type: goose.TypeGo},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), `invalid go migration: numeric component [1] in go migration does not match version in source "2_foo.sql"`)
	// Legacy functions.
	err = goose.SetGlobalMigrations(
		goose.Migration{Registered: true, Version: 1, UpFn: func(tx *sql.Tx) error { return nil }, Type: goose.TypeGo},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "invalid go migration: must not specify UpFn")
	err = goose.SetGlobalMigrations(
		goose.Migration{Registered: true, Version: 1, DownFn: func(tx *sql.Tx) error { return nil }, Type: goose.TypeGo},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "invalid go migration: must not specify DownFn")
	err = goose.SetGlobalMigrations(
		goose.Migration{Registered: true, Version: 1, UpFnNoTx: func(db *sql.DB) error { return nil }, Type: goose.TypeGo},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "invalid go migration: must not specify UpFnNoTx")
	err = goose.SetGlobalMigrations(
		goose.Migration{Registered: true, Version: 1, DownFnNoTx: func(db *sql.DB) error { return nil }, Type: goose.TypeGo},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "invalid go migration: must not specify DownFnNoTx")
	// Context-aware functions.
	err = goose.SetGlobalMigrations(
		goose.Migration{Registered: true, Version: 1, UpFnContext: fn, UpFnNoTxContext: fnNoTx, Type: goose.TypeGo},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "invalid go migration: must specify exactly one of UpFnContext or UpFnNoTxContext")
	err = goose.SetGlobalMigrations(
		goose.Migration{Registered: true, Version: 1, DownFnContext: fn, DownFnNoTxContext: fnNoTx, Type: goose.TypeGo},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "invalid go migration: must specify exactly one of DownFnContext or DownFnNoTxContext")
	// Source and version mismatch.
	err = goose.SetGlobalMigrations(
		goose.Migration{Registered: true, Version: 1, Source: "invalid_numeric.sql", Type: goose.TypeGo},
	)
	check.HasError(t, err)
	check.Contains(t, err.Error(), `invalid go migration: failed to parse version from migration file: invalid_numeric.sql`)
}
