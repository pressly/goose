package goose

import (
	"fmt"
)

var (
	registeredGoMigrations = make(map[int64]*Migration)
)

// ResetGlobalMigrations resets the global Go migrations registry.
//
// Not safe for concurrent use.
func ResetGlobalMigrations() {
	registeredGoMigrations = make(map[int64]*Migration)
}

// SetGlobalMigrations registers Go migrations globally. It returns an error if a migration with the
// same version has already been registered.
//
// Avoid constructing migrations manually, use [NewGoMigration] function.
//
// Source may be empty, but if it is set, it must be a path with a numeric component that matches
// the version. Do not register legacy non-context functions: UpFn, DownFn, UpFnNoTx, DownFnNoTx.
//
// Not safe for concurrent use.
func SetGlobalMigrations(migrations ...Migration) error {
	for _, m := range migrations {
		migration := &m
		if err := validGoMigration(migration); err != nil {
			return fmt.Errorf("invalid go migration: %w", err)
		}
		if err := verifyAndUpdateGoFunc(migration.goUp); err != nil {
			return fmt.Errorf("up function: %w", err)
		}
		if err := verifyAndUpdateGoFunc(migration.goDown); err != nil {
			return fmt.Errorf("down function: %w", err)
		}
		if err := updateLegacyFuncs(migration); err != nil {
			return fmt.Errorf("invalid go migration: %w", err)
		}
		if _, ok := registeredGoMigrations[m.Version]; ok {
			return fmt.Errorf("go migration with version %d already registered", m.Version)
		}
		m.Next, m.Previous = -1, -1 // Do not allow these to be set by the user.
		registeredGoMigrations[m.Version] = migration
	}
	return nil
}
