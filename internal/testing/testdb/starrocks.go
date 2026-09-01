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
	// Add explicit timeouts; StarRocks can accept TCP but drop MySQL protocol
	// connections during initialization which shows up as unexpected EOF.
	dsn += "&timeout=30s&readTimeout=30s&writeTimeout=30s"

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, cleanup, err
	}

	// Exponential backoff-retry, because the application in the container
	// might not be ready to accept connections yet. Add an extra sleep
	// because container take much longer to startup.
	pool.MaxWait = time.Minute * 4
	if err := pool.Retry(func() error {
		if err := db.Ping(); err != nil {
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

		// StarRocks FE may accept connections before any usable BE is available.
		// Goose migrations create OLAP tables which require a healthy backend.
		return starrocksWaitOlapReady(db)
	},
	); err != nil {
		return nil, cleanup, fmt.Errorf("could not connect to docker database: %v", err)
	}

	return db, cleanup, nil
}

func starrocksWaitOlapReady(db *sql.DB) error {
	const table = "__goose_backend_healthcheck"
	_, _ = db.Exec("DROP TABLE IF EXISTS " + table)

	// Minimal OLAP table; create will fail with "available backends: []" until BE is ready.
	create := `CREATE TABLE ` + table + ` (
		k1 INT
	) DUPLICATE KEY(k1)
	DISTRIBUTED BY HASH(k1) BUCKETS 1
	PROPERTIES ("replication_num" = "1")`

	if _, err := db.Exec(create); err != nil {
		return err
	}
	_, err := db.Exec("DROP TABLE IF EXISTS " + table)
	return err
}
