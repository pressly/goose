package provider

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/pressly/goose/v3/database"
	"github.com/pressly/goose/v3/lock"
)

const (
	// DefaultTablename is the default name of the database table used to track history of applied
	// migrations. It can be overridden using the [WithTableName] option when creating a new
	// provider.
	DefaultTablename = "goose_db_version"
)

// ProviderOption is a configuration option for a goose provider.
type ProviderOption interface {
	apply(*config) error
}

// WithStore configures the provider with a custom [database.Store] implementation.
//
// By default, the provider uses the [database.NewStore] function to create a store backed by the
// given dialect. However, this option allows users to provide their own implementation or call
// [database.NewStore] with custom options, such as setting the table name.
//
// Example:
//
//	// Create a store with a custom table name.
//	store, err := database.NewStore(database.DialectPostgres, "my_custom_table_name")
//	if err != nil {
//	    return err
//	}
//	// Create a provider with the custom store.
//	provider, err := goose.NewProvider("", db, nil, goose.WithStore(store))
//	if err != nil {
//	    return err
//	}
func WithStore(store database.Store) ProviderOption {
	return configFunc(func(c *config) error {
		if c.store != nil {
			return fmt.Errorf("store already set: %T", c.store)
		}
		if store == nil {
			return errors.New("store must not be nil")
		}
		if store.Tablename() == "" {
			return errors.New("store implementation must set the table name")
		}
		c.store = store
		return nil
	})
}

// WithVerbose enables verbose logging.
func WithVerbose(b bool) ProviderOption {
	return configFunc(func(c *config) error {
		c.verbose = b
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
// and down [GoMigration] may be nil. But if set, exactly one of Run or RunNoTx functions must be
// set.
func WithGoMigration(version int64, up, down *GoMigration) ProviderOption {
	return configFunc(func(c *config) error {
		if version < 1 {
			return errors.New("version must be greater than zero")
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
			up:   up,
			down: down,
		}
		return nil
	})
}

// WithAllowMissing allows the provider to apply missing (out-of-order) migrations.
//
// Example: migrations 1,3 are applied and then version 2,6 are introduced. If this option is true,
// then goose will apply 2 (missing) and 6 (new) instead of raising an error. The final order of
// applied migrations will be: 1,3,2,6. Out-of-order migrations are always applied first, followed
// by new migrations.
func WithAllowMissing(b bool) ProviderOption {
	return configFunc(func(c *config) error {
		c.allowMissing = b
		return nil
	})
}

// WithNoVersioning disables versioning. Disabling versioning allows applying migrations without
// tracking the versions in the database schema table. Useful for tests, seeding a database or
// running ad-hoc queries.
func WithNoVersioning(b bool) ProviderOption {
	return configFunc(func(c *config) error {
		c.noVersioning = b
		return nil
	})
}

type config struct {
	store database.Store

	verbose  bool
	excludes map[string]bool

	// Go migrations registered by the user. These will be merged/resolved with migrations from the
	// filesystem and init() functions.
	registered map[int64]*goMigration

	// Locking options
	lockEnabled   bool
	sessionLocker lock.SessionLocker

	// Feature
	noVersioning bool
	allowMissing bool
}

type configFunc func(*config) error

func (f configFunc) apply(cfg *config) error {
	return f(cfg)
}
