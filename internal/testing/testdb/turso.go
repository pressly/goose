package testdb

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

const (
	// ghcr.io/tursodatabase/libsql-server:v0.23.7
	TURSO_IMAGE   = "ghcr.io/tursodatabase/libsql-server"
	TURSO_VERSION = "v0.24.7"
	TURSO_PORT    = "8080"
)

// NewTurso starts a Turso docker container. Returns db connection and a docker cleanup function.
func NewTurso(options ...OptionsFunc) (db *sql.DB, cleanup func(), err error) {
	return newTurso(options...)
}

func newTurso(opts ...OptionsFunc) (*sql.DB, func(), error) {
	option := &options{}
	for _, f := range opts {
		f(option)
	}
	// Uses a sensible default on windows (tcp/http) and linux/osx (socket).
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, err
	}
	runOptions := &dockertest.RunOptions{
		Repository:   TURSO_IMAGE,
		Tag:          TURSO_VERSION,
		Labels:       map[string]string{"goose_test": "1"},
		PortBindings: make(map[docker.Port][]docker.PortBinding),
	}
	if option.debug {
		runOptions.Env = append(runOptions.Env, "RUST=trace")
	} else {
		runOptions.Env = append(runOptions.Env, "RUST=error")
	}
	if option.bindPort > 0 {
		runOptions.PortBindings[TURSO_PORT+"/tcp"] = []docker.PortBinding{
			{HostPort: strconv.Itoa(option.bindPort)},
		}
	}
	container, err := pool.RunWithOptions(
		runOptions,
		func(config *docker.HostConfig) {
			// Set AutoRemove to true so that stopped container goes away by itself.
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		},
	)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		if option.debug {
			// User must manually delete the Docker container.
			return
		}
		if err := pool.Purge(container); err != nil {
			log.Printf("failed to purge resource: %v", err)
		}
	}
	// Fetch port assigned to container

	var db *sql.DB
	// Exponential backoff-retry, because the application in the container
	// might not be ready to accept connections yet.
	if err := pool.Retry(func() error {
		db, err = tursoOpenDB(container)
		return err
	}); err != nil {
		return nil, cleanup, fmt.Errorf("could not connect to docker database: %w", err)
	}
	return db, cleanup, nil
}

func tursoOpenDB(container *dockertest.Resource) (*sql.DB, error) {
	address := fmt.Sprintf("http://127.0.0.1:%s", container.GetPort(TURSO_PORT+"/tcp"))
	db, err := sql.Open("libsql", address)
	if err != nil {
		return db, err
	}
	// let's do a ping to be sure we are connected
	var result int
	err = db.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		return nil, err
	}
	return db, nil
}
