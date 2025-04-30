package testdb

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const (
	// https://hub.docker.com/r/starrocks/allin1-ubuntu
	STARROCKS_IMAGE   = "starrocks/allin1-ubuntu"
	STARROCKS_VERSION = "3.2-latest"

	STARROCKS_USER    = "root"
	STARROCKS_INIT_DB = "migrations"
)

func newStarrocks(opts ...OptionsFunc) (*sql.DB, func(), error) {
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
		Repository:   STARROCKS_IMAGE,
		Tag:          STARROCKS_VERSION,
		Labels:       map[string]string{"goose_test": "1"},
		PortBindings: make(map[docker.Port][]docker.PortBinding),
		ExposedPorts: []string{"9030/tcp"},
	}
	if option.bindPort > 0 {
		options.PortBindings[docker.Port("9030/tcp")] = []docker.PortBinding{
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
	dsn := fmt.Sprintf("%s:%s@(%s:%s)/%s?parseTime=true&interpolateParams=true",
		STARROCKS_USER,
		"",
		"localhost",
		container.GetPort("9030/tcp"), // Fetch port dynamically assigned to container,
		"",
	)
	var db *sql.DB

	// Exponential backoff-retry, because the application in the container
	// might not be ready to accept connections yet. Add an extra sleep
	// because container take much longer to startup.
	pool.MaxWait = time.Minute * 2
	if err := pool.Retry(func() error {
		var err error
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			return err
		}

		_, err = db.Exec("CREATE DATABASE IF NOT EXISTS " + STARROCKS_INIT_DB)
		if err != nil {
			return fmt.Errorf("could not create initial database: %v", err)
		}
		_, err = db.Exec("USE " + STARROCKS_INIT_DB)
		if err != nil {
			return fmt.Errorf("could not set default initial database: %v", err)
		}

		return db.Ping()
	},
	); err != nil {
		return nil, cleanup, fmt.Errorf("could not connect to docker database: %v", err)
	}

	return db, cleanup, nil
}
