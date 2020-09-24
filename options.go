package goose

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
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
// this can be used with packages like packr that support the http.FileSystem
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

func (opts options) listSQLFiles() ([]string, error) {
	out := []string{}
	file, err := opts.fileSystem.Open("/")
	if err != nil {
		return out, err
	}

	files, err := file.Readdir(-1)
	if err != nil {
		return out, err
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".sql") {
			out = append(out, f.Name())
		}
	}
	return out, err
}

func (opts options) listGOFiles() ([]string, error) {
	out := []string{}
	file, err := opts.fileSystem.Open("/")
	if err != nil {
		return out, err
	}

	files, err := file.Readdir(-1)
	if err != nil {
		return out, err
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".go") {
			out = append(out, f.Name())
		}
	}
	return out, err
}