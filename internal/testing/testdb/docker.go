package testdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/pressly/dockermanage"
)

const (
	// startTimeout covers the image pull and the container start.
	startTimeout = 10 * time.Minute
	// readyTimeout is the default time a container has to become ready.
	readyTimeout = time.Minute
	// removeTimeout is the time cleanup has to remove a container.
	removeTimeout = time.Minute
)

// startContainer starts a container and waits until ready returns nil. The cleanup function removes
// the container, unless the debug option is set. If startContainer fails, it removes the container.
func startContainer(
	option *options,
	image string,
	timeout time.Duration,
	ready dockermanage.ReadinessFunc,
	startOptions ...dockermanage.StartOption,
) (*dockermanage.Container, func(), error) {
	manager, err := dockermanage.New()
	if err != nil {
		return nil, nil, fmt.Errorf("connect to docker: %w", err)
	}
	startOptions = append(startOptions, dockermanage.WithLabel("goose_test", "1"))
	startCtx, cancelStart := context.WithTimeout(context.Background(), startTimeout)
	defer cancelStart()
	container, err := manager.Start(startCtx, image, startOptions...)
	if err != nil {
		return nil, nil, errors.Join(err, manager.Close())
	}
	readyCtx, cancelReady := context.WithTimeout(context.Background(), timeout)
	defer cancelReady()
	if err := container.WaitReady(readyCtx, ready); err != nil {
		return nil, nil, errors.Join(err, removeContainer(option, manager, container))
	}
	return container, cleanupFunc(option, manager, container), nil
}

// cleanupFunc returns a function that removes the container and logs an error.
func cleanupFunc(option *options, manager *dockermanage.Manager, container *dockermanage.Container) func() {
	return func() {
		if err := removeContainer(option, manager, container); err != nil {
			log.Printf("failed to remove container %s: %v", container.ID(), err)
		}
	}
}

// removeContainer removes the container, unless the debug option is set, and closes the manager.
func removeContainer(option *options, manager *dockermanage.Manager, container *dockermanage.Container) error {
	if option.debug {
		// The user must remove the container.
		return manager.Close()
	}
	ctx, cancel := context.WithTimeout(context.Background(), removeTimeout)
	defer cancel()
	return errors.Join(container.Remove(ctx), manager.Close())
}

// portOption publishes containerPort on the port from the bind port option, or on a port that
// Docker picks.
func portOption(option *options, containerPort int) dockermanage.StartOption {
	if option.bindPort > 0 {
		return dockermanage.WithPortBinding(containerPort, option.bindPort)
	}
	return dockermanage.WithPort(containerPort)
}

func applyOptions(opts []OptionsFunc) *options {
	option := &options{}
	for _, f := range opts {
		f(option)
	}
	return option
}

// openAndPing opens a database and pings it. It closes the database if the ping fails.
func openAndPing(ctx context.Context, driverName, dsn string) (*sql.DB, error) {
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, errors.Join(err, db.Close())
	}
	return db, nil
}
