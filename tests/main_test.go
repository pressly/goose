package postgres_test

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var (
	debug = flag.Bool("debug", false, "Debug traps the test suite: useful for debugging running containers")
)

var (
	dialectPostgres = "postgres"
	postgresOptions = &dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "14-alpine",
		Env: []string{
			"POSTGRES_USER=postgres",
			"POSTGRES_PASSWORD=password1",
			"POSTGRES_DB=testdb",
			"listen_addresses = '*'",
		},
		Labels: map[string]string{"goose_test": "1"},
	}
)

func TestMain(m *testing.M) {
	flag.Parse()

	exitCode := m.Run()
	// Useful for debugging test services.
	if *debug {
		sigs := make(chan os.Signal, 1)
		done := make(chan bool, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigs
			done <- true
		}()
		log.Printf("entering debug mode: must exit (CTRL+C) and cleanup containers manually. Exit code: %d)", exitCode)
		<-done
	}
	os.Exit(exitCode)
}

func newDockerDatabase(t *testing.T, dialect string, port int) (*sql.DB, error) {
	// Uses a sensible default on windows (tcp/http) and linux/osx (socket).
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to docker: %v", err)
	}

	var options *dockertest.RunOptions
	portBindings := make(map[docker.Port][]docker.PortBinding)
	switch dialect {
	case dialectPostgres:
		options = postgresOptions
		if port != 0 {
			portBindings[docker.Port("5432/tcp")] = []docker.PortBinding{
				{HostPort: strconv.Itoa(port)},
			}
		}
	default:
		return nil, fmt.Errorf("unsupported dialect: %v", dialect)
	}

	container, err := pool.RunWithOptions(
		options,
		func(config *docker.HostConfig) {
			// Set AutoRemove to true so that stopped container goes away by itself.
			config.AutoRemove = true
			config.PortBindings = portBindings
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker container: %v", err)
	}
	t.Cleanup(func() {
		if *debug {
			// User must manually delete the Docker container.
			return
		}
		if err := pool.Purge(container); err != nil {
			log.Printf("failed to purge resource: %v", err)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start resource: %w", err)
	}
	n := container.GetPort("5432/tcp")
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		"localhost", n, "postgres", "password1", "testdb")
	var db *sql.DB
	// Exponential backoff-retry, because the application in the container
	// might not be ready to accept connections yet.
	if err := pool.Retry(
		func() error {
			var err error
			db, err = sql.Open("postgres", psqlInfo)
			if err != nil {
				return err
			}
			return db.Ping()
		},
	); err != nil {
		return nil, fmt.Errorf("could not connect to docker: %w", err)
	}
	return db, nil
}
