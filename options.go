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
	Logger   Logger
	Verbose  bool
	LockMode LockMode

	// Features.
	AllowMissing bool
	NoVersioning bool

	// Development.
	Debug            bool
	ExcludeFilenames []string

	// Unimplemented.
	//
	// See run_grouped.go for more details.
	groupedMigrations bool //nolint:golint,unused
}

func DefaultOptions() Options {
	return Options{
		TableName:  defaultTableName,
		Dir:        defaultDir,
		Filesystem: osFS{},
		Logger:     &stdLogger{},
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

// SetExcludeFilenames returns a new Options value with ExcludeFilenames set to the given value.
// ExcludeFilenames is a list of filenames to exclude when reading (and parsing) migrations from the
// filesystem. This is useful for skipping migrations in tests or development.
//
// Default: include all migrations.
func (o Options) SetExcludeFilenames(filenames ...string) Options {
	o.ExcludeFilenames = filenames
	return o
}

type LockMode int

const (
	LockModeNone LockMode = iota
	LockModeAdvisorySession
	// LockModeAdvisoryTransaction
)

func (l LockMode) String() string {
	switch l {
	case LockModeNone:
		return "none"
	case LockModeAdvisorySession:
		return "advisory-session"
	// case LockModeAdvisoryTransaction:
	// 	return "advisory-transaction"
	default:
		return "unknown"
	}
}

// SetLockMode returns a new Options value with LockMode set to the given value. LockMode is the
// locking mode to use when applying migrations. Locking is used to prevent multiple instances of
// goose from applying migrations concurrently.
//
// IMPORTANT: Locking is currently only supported for postgres. If you'd like to see support for
// other databases, please file an issue.
//
// Default: LockModeNone
func (o Options) SetLockMode(m LockMode) Options {
	o.LockMode = m
	return o
}

// setGroupedMigrations returns a new Options value with GroupedMigrations set to the given value.
// GroupedMigrations enables the ability to share a single transaction across multiple migrations.
//
// For more information, see: TODO(mf): add link to docs
//
// For example, say we have 6 new migrations to apply: 11,12,13,14,15,16. But migration 14 is marked
// with -- +goose NO TRANSACTION. Then the migrations will be applied sequentially in 3 groups:
//
//  1. migrations 11,12,13 will be applied in a single transaction and committed
//  2. migration 14 will be applied outside transaction and committed
//  3. migrations 15,16 will be applied in a single transaction and committed
//
// This feature is useful to avoid leaving the database in a partially migrated state. But, keep in
// mind there may be performance implications if you have a large number of migrations.
//
// Default: false
func (o Options) setGroupedMigrations(b bool) Options { //nolint:golint,unused
	o.groupedMigrations = b
	return o
}
