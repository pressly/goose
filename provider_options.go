package goose

import (
	"fmt"
	"io/fs"
)

const (
	defaultTablename = "goose_db_version"
	defaultDir       = "migrations"
)

// ProviderOptions is used to configure a Provider.
type ProviderOptions struct {
	// Dir is the directory where the migration files are located.
	//
	// Default: "migrations"
	Dir string
	// Tablename is the name of the database table used to track history of applied migrations.
	//
	// Default: "goose_db_version"
	Tablename string
	// Filesystem is the filesystem used to read the migration files.
	// Required field.
	//
	// Default: read from disk
	Filesystem fs.FS

	Verbose bool

	// Features
	AllowMissing bool
}

// DefaultOptions returns the default ProviderOptions.
func DefaultOptions() *ProviderOptions {
	return &ProviderOptions{
		Dir:        defaultDir,
		Tablename:  defaultTablename,
		Filesystem: osFS{},
	}
}

func validateOptions(opts *ProviderOptions) error {
	if opts.Dir == "" {
		return fmt.Errorf("dir must not be empty")
	}
	if opts.Tablename == "" {
		return fmt.Errorf("table must not be empty")
	}
	if opts.Filesystem == nil {
		return fmt.Errorf("filesystem must not be nil")
	}
	return nil
}
