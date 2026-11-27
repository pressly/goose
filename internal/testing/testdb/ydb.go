package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/pressly/dockermanage"
	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/balancers"
	ydblog "github.com/ydb-platform/ydb-go-sdk/v3/log"
	"github.com/ydb-platform/ydb-go-sdk/v3/trace"
)

const (
	YDB_IMAGE    = "ghcr.io/ydb-platform/local-ydb"
	YDB_VERSION  = "24.1"
	YDB_PORT     = 2136
	YDB_UI_PORT  = 8765
	YDB_DATABASE = "local"
)

func newYdb(opts ...OptionsFunc) (*sql.DB, func(), error) {
	option := applyOptions(opts)
	logLevel := "ERROR"
	if option.debug {
		logLevel = "NOTICE"
	}
	var db *sql.DB
	ready := func(ctx context.Context, container *dockermanage.Container) (err error) {
		// Run the health check of the image.
		result, err := container.Exec(ctx, []string{"sh", "./health_check"})
		if err != nil {
			return err
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("health check exit code %d", result.ExitCode)
		}
		addr, err := container.Addr(YDB_PORT)
		if err != nil {
			return err
		}
		dsn := fmt.Sprintf("grpc://%s/%s", addr, YDB_DATABASE)

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

		candidate := sql.OpenDB(connector)
		candidate.SetMaxIdleConns(5)
		candidate.SetMaxOpenConns(10)
		candidate.SetConnMaxLifetime(time.Hour)

		if err := candidate.PingContext(ctx); err != nil {
			return err
		}
		db = candidate
		return nil
	}
	_, cleanup, err := startContainer(option, YDB_IMAGE+":"+YDB_VERSION, readyTimeout, ready,
		portOption(option, YDB_PORT),
		dockermanage.WithEnv("YDB_USE_IN_MEMORY_PDISKS", "true"),
		dockermanage.WithEnv("YDB_LOCAL_SURVIVE_RESTART", "true"),
		dockermanage.WithEnv("GRPC_PORT", strconv.Itoa(YDB_PORT)),
		dockermanage.WithEnv("MON_PORT", strconv.Itoa(YDB_UI_PORT)),
		dockermanage.WithEnv("YDB_DEFAULT_LOG_LEVEL", logLevel),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("could not connect to docker database: %w", err)
	}
	return db, cleanup, nil
}
