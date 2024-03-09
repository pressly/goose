package testdb

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const (
	// https://hub.docker.com/_/postgres
	POSTGRES_IMAGE   = "postgres"
	POSTGRES_VERSION = "16-alpine"

	POSTGRES_DB       = "testdb"
	POSTGRES_USER     = "postgres"
	POSTGRES_PASSWORD = "password1"
)

func newPostgres(opts ...OptionsFunc) (*sql.DB, func(), error) {
	option := &options{}
	for _, f := range opts {
		f(option)
	}
	// Uses a sensible default on windows (tcp/http) and linux/osx (socket).
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to docker: %v", err)
	}
	options := &dockertest.RunOptions{
		Repository: POSTGRES_IMAGE,
		Tag:        POSTGRES_VERSION,
		Env: []string{
			"POSTGRES_USER=" + POSTGRES_USER,
			"POSTGRES_PASSWORD=" + POSTGRES_PASSWORD,
			"POSTGRES_DB=" + POSTGRES_DB,
			"listen_addresses = '*'",
		},
		Labels:       map[string]string{"goose_test": "1"},
		PortBindings: make(map[docker.Port][]docker.PortBinding),
	}
	if option.bindPort > 0 {
		options.PortBindings[docker.Port("5432/tcp")] = []docker.PortBinding{
			{HostPort: strconv.Itoa(option.bindPort)},
		}
	}
	container, err := pool.RunWithOptions(
		options,
		func(config *docker.HostConfig) {
			// Set AutoRemove to true so that stopped container goes away by itself.
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create docker container: %v", err)
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
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		"localhost",
		container.GetPort("5432/tcp"), // Fetch port dynamically assigned to container
		POSTGRES_USER,
		POSTGRES_PASSWORD,
		POSTGRES_DB,
	)
	var db *sql.DB
	// Exponential backoff-retry, because the application in the container
	// might not be ready to accept connections yet.
	if err := pool.Retry(
		func() error {
			var err error
			db, err = sql.Open("pgx", psqlInfo)
			if err != nil {
				return err
			}
			return db.Ping()
		},
	); err != nil {
		return nil, cleanup, fmt.Errorf("could not connect to docker database: %v", err)
	}
	return db, cleanup, nil
}
