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