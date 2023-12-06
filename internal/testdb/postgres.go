package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/docker/docker/api/types/container"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	// https://hub.docker.com/_/postgres
	POSTGRES_IMAGE = "postgres:16-alpine"

	POSTGRES_DB       = "testdb"
	POSTGRES_USER     = "postgres"
	POSTGRES_PASSWORD = "password1"
)

func newPostgres(opts ...OptionsFunc) (*sql.DB, func(), error) {
	var opt options
	for _, f := range opts {
		f(&opt)
	}

	ctx := context.Background()
	container, err := postgres.RunContainer(
		ctx,
		// Postgres options.
		postgres.WithDatabase(POSTGRES_DB),
		postgres.WithUsername(POSTGRES_USER),
		postgres.WithPassword(POSTGRES_PASSWORD),
		// Testcontainers options.
		testcontainers.WithImage(POSTGRES_IMAGE),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithPollInterval(500*time.Millisecond).
			WithStartupTimeout(5*time.Second),
		),
		testcontainers.WithHostConfigModifier(func(c *container.HostConfig) {
			c.AutoRemove = true
			c.RestartPolicy = container.RestartPolicy{Name: "no"}
		}),
		testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Labels: map[string]string{defaultLabel: "1"},
			},
		}),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start postgres container: %w", err)
	}
	connString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, maybeCleanup(container), fmt.Errorf("failed to get connection string: %w", err)
	}
	fmt.Fprintf(os.Stdout, "postgres container started: %s\n", connString)
	db, err := sql.Open("pgx", connString)
	if err != nil {
		return nil, maybeCleanup(container), fmt.Errorf("failed to connect to postgres: %w", err)
	}
	return db, maybeCleanup(container), nil
}
