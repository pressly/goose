package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/googleapis/go-sql-spanner" // Spanner driver
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"

	database "cloud.google.com/go/spanner/admin/database/apiv1"
	dbpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	inspb "cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
)

const (
	SPANNER_IMAGE   = "gcr.io/cloud-spanner-emulator/emulator"
	SPANNER_VERSION = "latest"

	SPANNER_PROJECT  = "test-project"
	SPANNER_INSTANCE = "test-instance"
	SPANNER_DATABASE = "test-db"
)

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
				"9010/tcp", "9020/tcp",
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
	os.Setenv("SPANNER_EMULATOR_HOST", emulatorHost)

	// Provision instance + database inside emulator
	err = pool.Retry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		if err := createSpannerResources(ctx); err != nil {
			return err
		}

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
		return nil, nil, fmt.Errorf("could not initialize spanner emulator: %v", err)
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

func createSpannerResources(ctx context.Context) error {
	instClient, err := instance.NewInstanceAdminClient(ctx)
	if err != nil {
		return fmt.Errorf("create instance client failed: %w", err)
	}
	defer instClient.Close()

	dbClient, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		return fmt.Errorf("create database client failed: %w", err)
	}
	defer dbClient.Close()

	instReq := &inspb.CreateInstanceRequest{
		Parent:     "projects/" + SPANNER_PROJECT,
		InstanceId: SPANNER_INSTANCE,
		Instance: &inspb.Instance{
			Config:      "projects/" + SPANNER_PROJECT + "/instanceConfigs/emulator-config",
			DisplayName: "Test Instance",
			NodeCount:   1,
		},
	}
	if _, err = instClient.CreateInstance(ctx, instReq); err != nil &&
		!strings.Contains(err.Error(), "AlreadyExists") {
		return fmt.Errorf("create instance failed: %w", err)
	}

	dbReq := &dbpb.CreateDatabaseRequest{
		Parent:          fmt.Sprintf("projects/%s/instances/%s", SPANNER_PROJECT, SPANNER_INSTANCE),
		CreateStatement: "CREATE DATABASE `" + SPANNER_DATABASE + "`",
	}
	if _, err = dbClient.CreateDatabase(ctx, dbReq); err != nil &&
		!strings.Contains(err.Error(), "AlreadyExists") {
		return fmt.Errorf("create database failed: %w", err)
	}

	return nil
}
