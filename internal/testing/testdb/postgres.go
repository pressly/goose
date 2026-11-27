package testdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/dockermanage"
	"github.com/pressly/dockermanage/postgres"
)

const (
	// https://hub.docker.com/_/postgres
	POSTGRES_IMAGE   = "postgres"
	POSTGRES_VERSION = "16-alpine"

	POSTGRES_DB       = "testdb"
	POSTGRES_USER     = "postgres"
	POSTGRES_PASSWORD = "password1"
)

func newPostgres(opts ...OptionsFunc) (*sql.DB, func(), error) {
	option := applyOptions(opts)
	manager, err := dockermanage.New()
	if err != nil {
		return nil, nil, fmt.Errorf("connect to docker: %w", err)
	}
	startOptions := []postgres.StartOption{
		postgres.WithImage(POSTGRES_IMAGE + ":" + POSTGRES_VERSION),
		postgres.WithDatabase(POSTGRES_DB),
		postgres.WithUser(POSTGRES_USER),
		postgres.WithPassword(POSTGRES_PASSWORD),
		postgres.WithLabel("goose_test", "1"),
	}
	if option.bindPort > 0 {
		startOptions = append(startOptions, postgres.WithHostPort(option.bindPort))
	}
	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()
	instance, err := postgres.Start(ctx, manager, startOptions...)
	if err != nil {
		return nil, nil, errors.Join(err, manager.Close())
	}
	db, err := sql.Open("pgx", instance.URL())
	if err != nil {
		return nil, nil, errors.Join(err, removeContainer(option, manager, instance.Container()))
	}
	return db, cleanupFunc(option, manager, instance.Container()), nil
}
