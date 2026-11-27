package testdb

import (
	"context"
	"database/sql"
	"errors"

	"github.com/pressly/dockermanage"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

const (
	// ghcr.io/tursodatabase/libsql-server:v0.23.7
	TURSO_IMAGE   = "ghcr.io/tursodatabase/libsql-server"
	TURSO_VERSION = "v0.24.7"
	TURSO_PORT    = 8080
)

// NewTurso starts a Turso docker container. Returns db connection and a docker cleanup function.
func NewTurso(options ...OptionsFunc) (db *sql.DB, cleanup func(), err error) {
	return newTurso(options...)
}

func newTurso(opts ...OptionsFunc) (*sql.DB, func(), error) {
	option := applyOptions(opts)
	logLevel := "error"
	if option.debug {
		logLevel = "trace"
	}
	var db *sql.DB
	ready := func(ctx context.Context, container *dockermanage.Container) error {
		addr, err := container.Addr(TURSO_PORT)
		if err != nil {
			return err
		}
		candidate, err := sql.Open("libsql", "http://"+addr)
		if err != nil {
			return err
		}
		// let's do a query to be sure we are connected
		var result int
		if err := candidate.QueryRowContext(ctx, "SELECT 1").Scan(&result); err != nil {
			return errors.Join(err, candidate.Close())
		}
		db = candidate
		return nil
	}
	_, cleanup, err := startContainer(option, TURSO_IMAGE+":"+TURSO_VERSION, readyTimeout, ready,
		portOption(option, TURSO_PORT),
		dockermanage.WithEnv("RUST", logLevel),
	)
	if err != nil {
		return nil, nil, err
	}
	return db, cleanup, nil
}
