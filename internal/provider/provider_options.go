package provider

import (
	"errors"
	"fmt"

	"github.com/pressly/goose/v3/lock"
)

const (
	defaultTablename = "goose_db_version"
)

// ProviderOption is a configuration option for a goose provider.
type ProviderOption interface {
	apply(*config) error
}

// WithTableName sets the name of the database table used to track history of applied migrations.
//
// If WithTableName is not called, the default value is "goose_db_version".
func WithTableName(name string) ProviderOption {
	return configFunc(func(c *config) error {
		if c.tableName != "" {
			return fmt.Errorf("table already set to %q", c.tableName)
		}
		if name == "" {
			return errors.New("table must not be empty")
		}
		c.tableName = name
		return nil
	})
}

// WithVerbose enables verbose logging.
func WithVerbose() ProviderOption {
	return configFunc(func(c *config) error {
		c.verbose = true
		return nil
	})
}

// WithSessionLocker enables locking using the provided SessionLocker.
//
// If WithSessionLocker is not called, locking is disabled.
func WithSessionLocker(locker lock.SessionLocker) ProviderOption {
	return configFunc(func(c *config) error {
		if c.lockEnabled {
			return errors.New("lock already enabled")
		}
		if c.sessionLocker != nil {
			return errors.New("session locker already set")
		}
		if locker == nil {
			return errors.New("session locker must not be nil")
		}
		c.lockEnabled = true
		c.sessionLocker = locker
		return nil
	})
}

type config struct {
	tableName string
	verbose   bool

	lockEnabled   bool
	sessionLocker lock.SessionLocker
}

type configFunc func(*config) error

func (f configFunc) apply(cfg *config) error {
	return f(cfg)
}
