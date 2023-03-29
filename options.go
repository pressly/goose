package goose

import "io/fs"

// Note: when adding a new option make sure to also add SetX method on Options. This enables us to
// have a clean Options struct with dedicated documentation for each option.

const (
	defaultTableName = "goose_db_version"
	defaultDir       = "./migrations"
)

// Options is a set of options to use when creating a new Provider.
//
// Options can be created with DefaultOptions and then modified with SetX methods. For example:
//
//	options := goose.DefaultOptions().SetDir("data/schema/migrations").SetVerbose(true)
//	goose.NewProvider(goose.DialectPostgres, db, options)
//
// Each option X is also documented on the SetX method.
type Options struct {
	// Required options.
	Dir        string
	TableName  string
	Filesystem fs.FS

	// Commonly modified options.
	Logger  Logger
	Verbose bool

	// Features.
	AllowMissing bool
	NoVersioning bool

	// Development.
	Debug           bool
	ExcludeVersions map[int64]bool

	// Unimplemented.
	lazyParsing bool
}

func DefaultOptions() Options {
	return Options{
		TableName:       defaultTableName,
		Dir:             defaultDir,
		Filesystem:      osFS{},
		Logger:          &stdLogger{},
		ExcludeVersions: make(map[int64]bool),
	}
}

// SetDir returns a new Options value with Dir set to the given value. Dir is the directory
// containing the migrations.
//
// Default: ./migrations
func (o Options) SetDir(s string) Options {
	o.Dir = s
	return o
}

// SetTableName returns a new Options value with TableName set to the given value. TableName is the
// database schema table used to record migrations.
//
// Default: goose_db_version
func (o Options) SetTableName(s string) Options {
	o.TableName = s
	return o
}

// SetFilesystem returns a new Options value with Filesystem set to the given value. Filesystem is
// the filesystem to use for reading migrations.
//
// Default: read from disk.
func (o Options) SetFilesystem(f fs.FS) Options {
	o.Filesystem = f
	return o
}

// SetLogger returns a new Options value with Logger set to the given value.
//
// Default: log to stderr if verbose is true.
func (o Options) SetLogger(l Logger) Options {
	o.Logger = l
	return o
}

// SetAllowMissing returns a new Options value with AllowMissing set to the given value.
// AllowMissing enables the ability to apply missing (out-of-order) migrations.
//
// Example: migrations 1,4 are applied and then version 2,3,5 are introduced. If this option is
// true, then goose will apply 2,3,5 instead of raising an error. The final order of applied
// migrations will be: 1,4,2,3,5.
//
// Default: false
func (o Options) SetAllowMissing(b bool) Options {
	o.AllowMissing = b
	return o
}

// SetVerbose returns a new Options value with Verbose set to the given value. Verbose prints
// additional information.
//
// Default: false
func (o Options) SetVerbose(b bool) Options {
	o.Verbose = b
	return o
}

// SetNoVersioning returns a new Options value with NoVersioning set to the given value.
// NoVersioning enables the ability to apply migrations without tracking the versions in the
// database schema table. Useful for seeding a database or running ad-hoc migrations.
//
// Default: false
func (o Options) SetNoVersioning(b bool) Options {
	o.NoVersioning = b
	return o
}

// SetDebug returns a new Options value with Debug set to the given value. Debug enables additional
// debugging information and is not intended to be used by end users. This is useful for debugging
// goose itself. For debugging migrations, use SetVerbose.
//
// Default: false
func (o Options) SetDebug(b bool) Options {
	o.Debug = b
	return o
}

// SetExcludeVersions returns a new Options value with ExcludeVersions set to the given value.
// ExcludeVersions is a list of migration versions to exclude when reading migrations from
// the filesystem. This is useful for skipping migrations in tests or development.
//
// Default: include all migrations.
func (o Options) SetExcludeVersions(versions []int64) Options {
	excluded := make(map[int64]bool, len(versions))
	for _, v := range versions {
		excluded[v] = true
	}
	o.ExcludeVersions = excluded
	return o
}
