package goose

import (
	"database/sql"
	"fmt"
	"net/http"
)

type options struct {
	dir string
	db	*sql.DB
	lockDB	bool
	fileSystem http.FileSystem
}

type Option func(*options)

func NewOptions(dir string, db *sql.DB, opts ...Option) *options {
	o := &options{
		dir: dir,
		db: db,
	}
	if dir != "" {
		o.fileSystem = http.Dir(dir)
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// WithFileSystem overrides the default fs with a different implementation
// this can be used with packages like vfsgen that support the http.FileSystem
// interface
func WithFileSystem(fs http.FileSystem) Option {
	return func(opts *options) {
		opts.fileSystem = fs
	}
}

// WithLockDB will attempt an exclusive lock on the migration table to keep other
// migrations from running until it's complete. There may not be a migration table
// if this is the initial migration. This will not raise an error it will continue
// migration without a lock.
func WithLockDB(lockDB bool) Option {
	return func(opts *options) {
		opts.lockDB = lockDB
		fmt.Println("LockDB")
	}
}