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
	cfg := &config{
		dir: dir,
		db: db,
	}
	if dir != "" {
		cfg.fileSystem = http.Dir(dir)
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

