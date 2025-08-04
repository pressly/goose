package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/googleapis/go-sql-spanner" // Spanner driver
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const (
	SPANNER_IMAGE   = "gcr.io/cloud-spanner-emulator/emulator"
	SPANNER_VERSION = "latest"

	SPANNER_PROJECT  = "test-project"
	SPANNER_INSTANCE = "test-instance"
	SPANNER_DATABASE = "test-db"
)

// newSpanner spins up a Cloud Spanner emulator and connects to it using the Go SQL driver.
func newSpanner(opts ...OptionsFunc) (*sql.DB, func(), error) {
	option := &options{}
	for _, f := range opts {
		f(option)
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to docker: %v", err)
	}

	resource, err := pool.RunWithOptions(
		&dockertest.RunOptions{
			Repository: SPANNER_IMAGE,
			Tag:        SPANNER_VERSION,
			ExposedPorts: []string{
				"9010/tcp",
			},
			Labels: map[string]string{"goose_test": "1"},
		},
		func(config *docker.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start spanner emulator container: %v", err)
	}

	hostPort := resource.GetPort("9010/tcp")
	emulatorHost := fmt.Sprintf("localhost:%s", hostPort)

	// Set environment variable so that the Spanner driver connects to the emulator.
	os.Setenv("SPANNER_EMULATOR_HOST", emulatorHost)

	// Use gcloud CLI or spanner admin client to create instance & database.
	err = pool.Retry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		dsn := fmt.Sprintf("projects/%s/instances/%s/databases/%s", SPANNER_PROJECT, SPANNER_INSTANCE, SPANNER_DATABASE)
		db, err := sql.Open("spanner", dsn)
		if err != nil {
			return err
		}
		defer db.Close()

		return db.PingContext(ctx)
	})
	if err != nil {
		_ = pool.Purge(resource)
		return nil, nil, fmt.Errorf("could not ping spanner emulator: %v", err)
	}

	dsn := fmt.Sprintf("projects/%s/instances/%s/databases/%s", SPANNER_PROJECT, SPANNER_INSTANCE, SPANNER_DATABASE)
	db, err := sql.Open("spanner", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open spanner DB: %v", err)
	}

	cleanup := func() {
		if err := db.Close(); err != nil {
			log.Printf("failed to close spanner db: %v", err)
		}
		if err := pool.Purge(resource); err != nil {
			log.Printf("failed to purge spanner emulator container: %v", err)
		}
	}

	return db, cleanup, nil
}
