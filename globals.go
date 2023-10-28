package goose

import (
	"errors"
	"fmt"
)

var (
	registeredGoMigrations = make(map[int64]*Migration)
)

// ResetGlobalMigrations resets the global go migrations registry.
//
// Not safe for concurrent use.
func ResetGlobalMigrations() {
	registeredGoMigrations = make(map[int64]*Migration)
}

// SetGlobalMigrations registers go migrations globally. It returns an error if a migration with the
// same version has already been registered.
//
// Source may be empty, but if it is set, it must be a path with a numeric component that matches
// the version. Do not register legacy non-context functions: UpFn, DownFn, UpFnNoTx, DownFnNoTx.
//
// Not safe for concurrent use.
func SetGlobalMigrations(migrations ...Migration) error {
	for _, m := range migrations {
		// make a copy of the migration so we can modify it without affecting the original.
		if err := validGoMigration(&m); err != nil {
			return fmt.Errorf("invalid go migration: %w", err)
		}
		if _, ok := registeredGoMigrations[m.Version]; ok {
			return fmt.Errorf("go migration with version %d already registered", m.Version)
		}
		m.Next, m.Previous = -1, -1 // Do not allow these to be set by the user.
		registeredGoMigrations[m.Version] = &m
	}
	return nil
}

func validGoMigration(m *Migration) error {
	if m == nil {
		return errors.New("must not be nil")
	}
	if !m.Registered {
		return errors.New("must be registered")
	}
	if m.Type != TypeGo {
		return fmt.Errorf("type must be %q", TypeGo)
	}
	if m.Version < 1 {
		return errors.New("version must be greater than zero")
	}
	if m.Source != "" {
		// If the source is set, expect it to be a path with a numeric component that matches the
		// version. This field is not intended to be used for descriptive purposes.
		version, err := NumericComponent(m.Source)
		if err != nil {
			return err
		}
		if version != m.Version {
			return fmt.Errorf("numeric component [%d] in go migration does not match version in source %q", m.Version, m.Source)
		}
	}
	// It's valid for all of these funcs to be nil. Which means version the go migration but do not
	// run anything.
	if m.UpFnContext != nil && m.UpFnNoTxContext != nil {
		return errors.New("must specify exactly one of UpFnContext or UpFnNoTxContext")
	}
	if m.DownFnContext != nil && m.DownFnNoTxContext != nil {
		return errors.New("must specify exactly one of DownFnContext or DownFnNoTxContext")
	}
	// Do not allow legacy functions to be set.
	if m.UpFn != nil {
		return errors.New("must not specify UpFn")
	}
	if m.DownFn != nil {
		return errors.New("must not specify DownFn")
	}
	if m.UpFnNoTx != nil {
		return errors.New("must not specify UpFnNoTx")
	}
	if m.DownFnNoTx != nil {
		return errors.New("must not specify DownFnNoTx")
	}
	return nil
}
