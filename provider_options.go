package goose

import (
	"errors"
	"fmt"
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

type config struct {
	tableName string
	verbose   bool
}

type configFunc func(*config) error

func (o configFunc) apply(cfg *config) error {
	return o(cfg)
}
