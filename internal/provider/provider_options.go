package provider

import (
	"context"
	"database/sql"
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

// WithExcludes excludes the given file names from the list of migrations.
//
// If WithExcludes is called multiple times, the list of excludes is merged.
func WithExcludes(excludes []string) ProviderOption {
	return configFunc(func(c *config) error {
		for _, name := range excludes {
			c.excludes[name] = true
		}
		return nil
	})
}

// GoMigration is a user-defined Go migration, registered using the option [WithGoMigration].
type GoMigration struct {
	// One of the following must be set:
	Run func(context.Context, *sql.Tx) error
	// -- OR --
	RunNoTx func(context.Context, *sql.DB) error
}

// WithGoMigration registers a Go migration with the given version.
//
// If WithGoMigration is called multiple times with the same version, an error is returned. Both up
// and down functions may be nil. But if set, exactly one of Run or RunNoTx functions must be set.
func WithGoMigration(version int64, up, down *GoMigration) ProviderOption {
	return configFunc(func(c *config) error {
		if version < 1 {
			return fmt.Errorf("go migration version must be greater than 0")
		}
		if _, ok := c.registered[version]; ok {
			return fmt.Errorf("go migration with version %d already registered", version)
		}
		// Allow nil up/down functions. This enables users to apply "no-op" migrations, while
		// versioning them.
		if up != nil {
			if up.Run == nil && up.RunNoTx == nil {
				return fmt.Errorf("go migration with version %d must have an up function", version)
			}
			if up.Run != nil && up.RunNoTx != nil {
				return fmt.Errorf("go migration with version %d must not have both an up and upNoTx function", version)
			}
		}
		if down != nil {
			if down.Run == nil && down.RunNoTx == nil {
				return fmt.Errorf("go migration with version %d must have a down function", version)
			}
			if down.Run != nil && down.RunNoTx != nil {
				return fmt.Errorf("go migration with version %d must not have both a down and downNoTx function", version)
			}
		}
		c.registered[version] = &goMigration{
			version: version,
			up:      up,
			down:    down,
		}
		return nil
	})
}

type goMigration struct {
	version  int64
	up, down *GoMigration
}

type config struct {
	tableName string
	verbose   bool
	excludes  map[string]bool

	// Go migrations registered by the user. These will be merged/resolved with migrations from the
	// filesystem and init() functions.
	registered map[int64]*goMigration

	// Locking options
	lockEnabled   bool
	sessionLocker lock.SessionLocker
}

type configFunc func(*config) error

func (f configFunc) apply(cfg *config) error {
	return f(cfg)
}
