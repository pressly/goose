package goose

import "net/http"

// Option function that takes a config and sets a property on it
type Option func(*config)

// WithFileSystem overrides the default fs with a different implementation
// this can be used with packages like vfsgen that support the http.FileSystem
// interface
func WithFileSystem(fs http.FileSystem) Option {
	return func(opts *config) {
		opts.fileSystem = fs
	}
}

// WithLockDB will attempt an exclusive lock on the migration table to keep other
// migrations from running until it's complete. There may not be a migration table
// if this is the initial migration. This will not raise an error it will continue
// migration without a lock.
func WithLockDB(lockDB bool) Option {
	return func(opts *config) {
		opts.lockDB = lockDB
	}
}