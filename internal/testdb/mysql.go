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
	// https://hub.docker.com/_/mysql
	MYSQL_IMAGE   = "mysql"
	MYSQL_VERSION = "8"

	MYSQL_DB       = "testdb"
	MYSQL_USER     = "tester"
	MYSQL_PASSWORD = "password1"
)

func newMySQL(opts ...OptionsFunc) (*sql.DB, func(), error) {
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
		Repository: MYSQL_IMAGE,
		Tag:        MYSQL_VERSION,
		Env: []string{
			"MYSQL_USER=" + MYSQL_USER,
			"MYSQL_PASSWORD=" + MYSQL_PASSWORD,
			"MYSQL_ROOT_PASSWORD=" + MYSQL_PASSWORD,
			"MYSQL_DATABASE=" + MYSQL_DB,
		},
		Labels:       map[string]string{"goose_test": "1"},
		PortBindings: make(map[docker.Port][]docker.PortBinding),
	}
	if option.bindPort > 0 {
		options.PortBindings[docker.Port("3306/tcp")] = []docker.PortBinding{
			{HostPort: strconv.Itoa(option.bindPort)},
		}
	}
	container, err := pool.RunWithOptions(
		options,
		func(config *docker.HostConfig) {
			// Set AutoRemove to true so that stopped container goes away by itself.
			config.AutoRemove = true
			// config.PortBindings = options.PortBindings
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
	// MySQL DSN: username:password@protocol(address)/dbname?param=value
	dsn := fmt.Sprintf("%s:%s@(%s:%s)/%s?parseTime=true&multiStatements=true",
		MYSQL_USER,
		MYSQL_PASSWORD,
		"localhost",
		container.GetPort("3306/tcp"), // Fetch port dynamically assigned to container
		MYSQL_DB,
	)
	var db *sql.DB
	time.Sleep(5 * time.Second)
	if err := pool.Retry(func() error {
		var err error
		db, err = sql.Open("mysql", dsn)
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
