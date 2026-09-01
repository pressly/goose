package testdb

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const (
	// https://hub.docker.com/r/starrocks/allin1-ubuntu
	STARROCKS_IMAGE   = "starrocks/allin1-ubuntu"
	STARROCKS_VERSION = "3.5.11"

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

	// Exponential backoff-retry because the frontend can accept connections before a backend is
	// available. Creating a table requires at least one alive backend.
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

		if err := db.Ping(); err != nil {
			return err
		}
		return checkStarrocksBackend(db)
	},
	); err != nil {
		return nil, cleanup, fmt.Errorf("could not connect to docker database: %v", err)
	}

	return db, cleanup, nil
}

func checkStarrocksBackend(db *sql.DB) error {
	rows, err := db.Query("SHOW BACKENDS")
	if err != nil {
		return fmt.Errorf("could not query StarRocks backends: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("could not get StarRocks backend columns: %v", err)
	}
	aliveIndex := -1
	for i, column := range columns {
		if strings.EqualFold(column, "Alive") {
			aliveIndex = i
			break
		}
	}
	if aliveIndex == -1 {
		return fmt.Errorf("StarRocks backend status does not include an Alive column")
	}

	values := make([]sql.RawBytes, len(columns))
	dest := make([]any, len(columns))
	for i := range values {
		dest[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return fmt.Errorf("could not scan StarRocks backend status: %v", err)
		}
		if strings.EqualFold(string(values[aliveIndex]), "true") {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("could not read StarRocks backend status: %v", err)
	}
	return fmt.Errorf("no alive StarRocks backends")
}
