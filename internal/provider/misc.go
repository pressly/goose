package provider

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/pressly/goose/v3"
)

type MigrationCopy struct {
	Version                            int64
	Source                             string // path to .sql script or go file
	Registered                         bool
	UpFnContext, DownFnContext         func(context.Context, *sql.Tx) error
	UpFnNoTxContext, DownFnNoTxContext func(context.Context, *sql.DB) error
}

var registeredGoMigrations = make(map[int64]*MigrationCopy)

// SetGlobalGoMigrations registers the given go migrations globally. It returns an error if any of
// the migrations are nil or if a migration with the same version has already been registered.
//
// Not safe for concurrent use.
func SetGlobalGoMigrations(migrations []*MigrationCopy) error {
	for _, m := range migrations {
		if m == nil {
			return errors.New("cannot register nil go migration")
		}
		if m.Version < 1 {
			return errors.New("migration versions must be greater than zero")
		}
		if !m.Registered {
			return errors.New("migration must be registered")
		}
		if m.Source != "" {
			// If the source is set, expect it to be a file path with a numeric component that
			// matches the version.
			version, err := goose.NumericComponent(m.Source)
			if err != nil {
				return err
			}
			if version != m.Version {
				return fmt.Errorf("migration version %d does not match source %q", m.Version, m.Source)
			}
		}
		// It's valid for all of these to be nil.
		if m.UpFnContext != nil && m.UpFnNoTxContext != nil {
			return errors.New("must specify exactly one of UpFnContext or UpFnNoTxContext")
		}
		if m.DownFnContext != nil && m.DownFnNoTxContext != nil {
			return errors.New("must specify exactly one of DownFnContext or DownFnNoTxContext")
		}
		if _, ok := registeredGoMigrations[m.Version]; ok {
			return fmt.Errorf("go migration with version %d already registered", m.Version)
		}
		registeredGoMigrations[m.Version] = m
	}
	return nil
}

// ResetGlobalGoMigrations resets the global go migrations registry.
//
// Not safe for concurrent use.
func ResetGlobalGoMigrations() {
	registeredGoMigrations = make(map[int64]*MigrationCopy)
}
