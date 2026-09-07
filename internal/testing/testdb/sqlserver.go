package testdb

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const (
	// https://hub.docker.com/_/microsoft-mssql-server
	SQLSERVER_IMAGE   = "mcr.microsoft.com/mssql/server"
	SQLSERVER_VERSION = "2022-latest"

	SQLSERVER_DB       = "master"
	SQLSERVER_USER     = "sa"
	SQLSERVER_PASSWORD = "Password123!"
)

func newSqlserver(opts ...OptionsFunc) (*sql.DB, func(), error) {
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
		Repository: SQLSERVER_IMAGE,
		Tag:        SQLSERVER_VERSION,
		Env: []string{
			"ACCEPT_EULA=Y",
			"MSSQL_SA_PASSWORD=" + SQLSERVER_PASSWORD,
		},
		Labels:       map[string]string{"goose_test": "1"},
		PortBindings: make(map[docker.Port][]docker.PortBinding),
	}
	if option.bindPort > 0 {
		options.PortBindings[docker.Port("1433/tcp")] = []docker.PortBinding{
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
	connStr := fmt.Sprintf("sqlserver://%s:%s@localhost:%s?database=%s",
		SQLSERVER_USER,
		SQLSERVER_PASSWORD,
		container.GetPort("1433/tcp"), // Fetch port dynamically assigned to container
		SQLSERVER_DB,
	)
	var db *sql.DB
	// Exponential backoff-retry, because the application in the container
	// might not be ready to accept connections yet. SQL Server takes longer to start.
	if err := pool.Retry(
		func() error {
			var err error
			db, err = sql.Open("sqlserver", connStr)
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
