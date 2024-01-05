package main_test

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	main "go-pgx-migration"
	"log"
	"os"
	"testing"
	"time"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	dockerPool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker %v", err)
	}

	resource, err := dockerPool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "16",
		Env: []string{
			"POSTGRES_PASSWORD=secret",
			"POSTGRES_USER=postgres",
			"POSTGRES_DB=postgres",
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		log.Fatalf("Could not start resource %v", err)
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	databaseUrl := fmt.Sprintf("postgres://postgres:secret@%s/postgres?sslmode=disable", hostAndPort)

	if err := resource.Expire(120); err != nil {
		log.Fatalf("Could not set expire %v", err)
	} // Tell docker to hard kill the container in 120 seconds

	dockerPool.MaxWait = 120 * time.Second
	if err := dockerPool.Retry(func() error {
		var err error
		pool, err = pgxpool.New(context.Background(), databaseUrl)
		if err != nil {
			return err
		}
		return pool.Ping(context.Background())
	}); err != nil {
		log.Fatalf("Could not connect to docker %v", err)
	}
	os.Exit(m.Run())
}

func TestMigration(t *testing.T) {
	migration, err := main.NewMigration(pool)
	if err != nil {
		t.Errorf("NewMigration failed %v", err)
	}

	t.Run("Up Migration", func(t *testing.T) {
		err = migration.Up()
		if err != nil {
			t.Errorf("Migration failed %v", err)
		}
	})
	t.Run("Down Migration", func(t *testing.T) {
		err = migration.Down()
		if err != nil {
			t.Errorf("Migration failed %v", err)
		}
	})
}
