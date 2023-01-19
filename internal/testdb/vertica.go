package testdb

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	_ "github.com/vertica/vertica-sql-go"
)

const (
	// https://hub.docker.com/r/vertica/vertica-ce
	VERTICA_IMAGE   = "vertica/vertica-ce"
	VERTICA_VERSION = "12.0.0-0"
	VERTICA_DB      = "testdb"
)

func newVertica(opts ...OptionsFunc) (*sql.DB, func(), error) {
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
		Repository: VERTICA_IMAGE,
		Tag:        VERTICA_VERSION,
		Env: []string{
			"VERTICA_DB_NAME=" + VERTICA_DB,
			"VMART_ETL_SCRIPT=", // Don't install VMART data inside container.
		},
		Labels:       map[string]string{"goose_test": "1"},
		PortBindings: make(map[docker.Port][]docker.PortBinding),
		// Prevent package installation for faster container startup.
		Mounts: []string{"/tmp/empty:/opt/vertica/packages"},
	}
	if option.bindPort > 0 {
		options.PortBindings[docker.Port("5433/tcp")] = []docker.PortBinding{
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
	verticaInfo := fmt.Sprintf("vertica://%s:%s@%s:%s/%s",
		"dbadmin",
		"",
		"localhost",
		container.GetPort("5433/tcp"), // Fetch port dynamically assigned to container
		VERTICA_DB,
	)

	var db *sql.DB
	// Give vertica a head start since the container takes a little bit to start up.
	time.Sleep(time.Second * 15)

	// Exponential backoff-retry, because the application in the container
	// might not be ready to accept connections yet.
	if err := pool.Retry(
		func() error {
			var err error
			db, err = sql.Open("vertica", verticaInfo)
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
