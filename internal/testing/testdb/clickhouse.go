package testdb

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const (
	// https://hub.docker.com/r/clickhouse/clickhouse-server/
	CLICKHOUSE_IMAGE   = "clickhouse/clickhouse-server"
	CLICKHOUSE_VERSION = "24-alpine"

	CLICKHOUSE_DB                        = "clickdb"
	CLICKHOUSE_USER                      = "clickuser"
	CLICKHOUSE_PASSWORD                  = "password1"
	CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT = "1"
)

func newClickHouse(opts ...OptionsFunc) (*sql.DB, func(), error) {
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
		Repository: CLICKHOUSE_IMAGE,
		Tag:        CLICKHOUSE_VERSION,
		Env: []string{
			"CLICKHOUSE_DB=" + CLICKHOUSE_DB,
			"CLICKHOUSE_USER=" + CLICKHOUSE_USER,
			"CLICKHOUSE_PASSWORD=" + CLICKHOUSE_PASSWORD,
			"CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT=" + CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT,
		},
		Labels:       map[string]string{"goose_test": "1"},
		PortBindings: make(map[docker.Port][]docker.PortBinding),
	}
	// Port 8123 is used for HTTP, but we're using the TCP protocol endpoint (port 9000).
	// Ref: https://clickhouse.com/docs/en/interfaces/http/
	// Ref: https://clickhouse.com/docs/en/interfaces/tcp/
	if option.bindPort > 0 {
		runOptions.PortBindings[docker.Port("9000/tcp")] = []docker.PortBinding{
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
	address := fmt.Sprintf("%s:%s", "localhost", container.GetPort("9000/tcp"))

	var db *sql.DB
	// Exponential backoff-retry, because the application in the container
	// might not be ready to accept connections yet.
	if err := pool.Retry(func() error {
		db = clickHouseOpenDB(address, nil, option.debug)
		return db.Ping()
	}); err != nil {
		return nil, cleanup, fmt.Errorf("could not connect to docker database: %w", err)
	}
	return db, cleanup, nil
}

func clickHouseOpenDB(address string, tlsConfig *tls.Config, debug bool) *sql.DB {
	db := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{address},
		Auth: clickhouse.Auth{
			Database: CLICKHOUSE_DB,
			Username: CLICKHOUSE_USER,
			Password: CLICKHOUSE_PASSWORD,
		},
		TLS: tlsConfig,
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 5 * time.Second,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		Debug: debug,
	})
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(time.Hour)
	return db
}
