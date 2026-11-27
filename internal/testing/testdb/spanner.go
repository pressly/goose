package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/googleapis/go-sql-spanner" // Spanner driver
	"github.com/pressly/dockermanage"

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
	option := applyOptions(opts)
	dsn := fmt.Sprintf("projects/%s/instances/%s/databases/%s", SPANNER_PROJECT, SPANNER_INSTANCE, SPANNER_DATABASE)
	// Provision instance + database inside emulator
	ready := func(ctx context.Context, container *dockermanage.Container) error {
		emulatorHost, err := container.Addr(9010)
		if err != nil {
			return err
		}
		os.Setenv("SPANNER_EMULATOR_HOST", emulatorHost)
		if err := createSpannerResources(ctx); err != nil {
			return err
		}
		db, err := sql.Open("spanner", dsn)
		if err != nil {
			return err
		}
		defer db.Close()
		return db.PingContext(ctx)
	}
	_, cleanupContainer, err := startContainer(option, SPANNER_IMAGE+":"+SPANNER_VERSION, readyTimeout, ready,
		dockermanage.WithPort(9010),
		dockermanage.WithPort(9020),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("could not initialize spanner emulator: %w", err)
	}

	db, err := sql.Open("spanner", dsn)
	if err != nil {
		cleanupContainer()
		return nil, nil, fmt.Errorf("failed to open spanner DB: %v", err)
	}

	cleanup := func() {
		if err := db.Close(); err != nil {
			log.Printf("failed to close spanner db: %v", err)
		}
		cleanupContainer()
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
