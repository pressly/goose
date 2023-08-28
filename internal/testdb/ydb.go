package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/ory/dockertest/v3/docker/types"
	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/balancers"
)

const (
	YDB_IMAGE    = "cr.yandex/yc/yandex-docker-local-ydb"
	YDB_VERSION  = "23.1"
	YDB_PORT     = "2136"
	YDB_TLS_PORT = "2135"
	YDB_MON_PORT = "8765"
	YDB_DATABASE = "local"
)

func newYdbWIthNative(opts ...OptionsFunc) (*sql.DB, *ydb.Driver, func(), error) {
	option := &options{}
	for _, f := range opts {
		f(option)
	}
	// Uses a sensible default on windows (tcp/http) and linux/osx (socket).
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, nil, err
	}
	runOptions := &dockertest.RunOptions{
		Repository: YDB_IMAGE,
		Tag:        YDB_VERSION,
		Env: []string{
			"YDB_USE_IN_MEMORY_PDISKS=true",
			"GRPC_PORT=" + YDB_PORT,
			"GRPC_TLS_PORT=" + YDB_TLS_PORT,
			"GRPC_MON_PORT=" + YDB_MON_PORT,
		},
		Labels:       map[string]string{"goose_test": "1"},
		PortBindings: map[docker.Port][]docker.PortBinding{},
		Mounts:       []string{os.TempDir() + ":/ydb_certs"},
		Hostname:     "localhost",
	}
	if option.debug {
		runOptions.Env = append(runOptions.Env, "YDB_DEFAULT_LOG_LEVEL=NOTICE")
	} else {
		runOptions.Env = append(runOptions.Env, "YDB_DEFAULT_LOG_LEVEL=ERROR")
	}
	if option.bindPort > 0 {
		runOptions.PortBindings[docker.Port(YDB_PORT+"/tcp")] = []docker.PortBinding{
			{HostPort: strconv.Itoa(option.bindPort)},
		}
	}
	container, err := pool.RunWithOptions(
		runOptions,
		func(config *docker.HostConfig) {
			// Set AutoRemove to true so that stopped container goes away by itself.
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
			config.Init = true
		},
	)
	if err != nil {
		return nil, nil, nil, err
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
	// Fetch port assigned to container
	dsn := fmt.Sprintf("grpc://%s:%s/%s",
		"localhost",
		container.GetPort(YDB_PORT+"/tcp"),
		YDB_DATABASE,
	)

	var db *sql.DB
	var extraNativeDriver *ydb.Driver
	// Exponential backoff-retry, because the application in the container
	// might not be ready to accept connections yet.
	if err := pool.Retry(func() (err error) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		if err := waitInit(ctx, pool, container.Container.ID); err != nil {
			return err
		}

		opts := []ydb.Option{
			ydb.WithBalancer(balancers.SingleConn()),
		}

		//if option.debug {
		//	opts = append(opts, ydb.WithLogger(nil, trace.DetailsAll))
		//}

		extraNativeDriver, err = ydb.Open(ctx, dsn, opts...)
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				_ = extraNativeDriver.Close(context.Background())
			}
		}()

		nativeDriver, err := ydb.Open(ctx, dsn, opts...)
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				_ = nativeDriver.Close(context.Background())
			}
		}()
		connector, err := ydb.Connector(nativeDriver,
			ydb.WithDefaultQueryMode(ydb.ScriptingQueryMode),
			ydb.WithFakeTx(ydb.ScriptingQueryMode),
			ydb.WithAutoDeclare(),
			ydb.WithNumericArgs(),
		)
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				_ = connector.Close()
			}
		}()

		db = sql.OpenDB(connector)
		db.SetMaxIdleConns(5)
		db.SetMaxOpenConns(10)
		db.SetConnMaxLifetime(time.Hour)

		err = db.Ping()
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, nil, cleanup, fmt.Errorf("could not connect to docker database: %w", err)
	}
	return db, extraNativeDriver, cleanup, nil
}

func waitInit(ctx context.Context, pool *dockertest.Pool, id string) error {
	var (
		initDoneCh = make(chan struct{})
		initErr    error
	)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				attemptCtx, attemptCancel := context.WithTimeout(context.Background(), time.Second)
				status, err := getHealthStatus(attemptCtx, pool, id)
				attemptCancel()

				if err != nil {
					initDoneCh <- struct{}{}
					initErr = err
					return
				}

				if status == types.Healthy {
					initDoneCh <- struct{}{}
					return
				}
			}

		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-initDoneCh:
			return initErr
		}
	}
}

func getHealthStatus(ctx context.Context, pool *dockertest.Pool, id string) (string, error) {
	currentContainer, err := pool.Client.InspectContainerWithContext(id, ctx)
	if err != nil {
		return "", err
	}

	return currentContainer.State.Health.Status, nil

}
