package goose

import (
	"database/sql"
	"net/http"
)

type config struct {
	dir string
	db	*sql.DB
	lockDB	bool
	fileSystem http.FileSystem
}



func newConfig(dir string, db *sql.DB, opts ...Option) *config {
	o := &config{
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

