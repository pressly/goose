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
	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/balancers"
	ydblog "github.com/ydb-platform/ydb-go-sdk/v3/log"
	"github.com/ydb-platform/ydb-go-sdk/v3/trace"
)

const (
	YDB_IMAGE    = "ghcr.io/ydb-platform/local-ydb"
	YDB_VERSION  = "24.1"
	YDB_PORT     = "2136"
	YDB_UI_PORT  = "8765"
	YDB_DATABASE = "local"
)

func newYdb(opts ...OptionsFunc) (*sql.DB, func(), error) {
	option := &options{}
	for _, f := range opts {
		f(option)
	}
	// Uses a sensible default on windows (tcp/http) and linux/osx (socket).
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, err
	}
	runOptions := &dockertest.RunOptions{
		Repository: YDB_IMAGE,
		Tag:        YDB_VERSION,
		Env: []string{
			"YDB_USE_IN_MEMORY_PDISKS=true",
			"YDB_LOCAL_SURVIVE_RESTART=true",
			"GRPC_PORT=" + YDB_PORT,
			"MON_PORT=" + YDB_UI_PORT,
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
		runOptions.PortBindings[YDB_PORT+"/tcp"] = []docker.PortBinding{
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
		return nil, nil, err
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
	// Exponential backoff-retry, because the application in the container
	// might not be ready to accept connections yet.
	if err := pool.Retry(func() (err error) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		if err := containerWaitHealthy(ctx, pool, container.Container.ID); err != nil {
			return err
		}

		opts := []ydb.Option{
			ydb.WithBalancer(balancers.SingleConn()),
		}

		if option.debug {
			opts = append(opts, ydb.WithLogger(ydblog.Default(os.Stdout), trace.DetailsAll, ydblog.WithLogQuery()))
		}

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
		cleanup()
		return nil, nil, fmt.Errorf("could not connect to docker database: %w", err)
	}
	return db, cleanup, nil
}
