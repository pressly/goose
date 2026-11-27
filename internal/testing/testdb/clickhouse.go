package testdb

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/pressly/dockermanage"
)

const (
	// https://hub.docker.com/r/clickhouse/clickhouse-server/
	CLICKHOUSE_IMAGE   = "clickhouse/clickhouse-server"
	CLICKHOUSE_VERSION = "24-alpine"

	CLICKHOUSE_DB                        = "clickdb"
	CLICKHOUSE_USER                      = "clickuser"
	CLICKHOUSE_PASSWORD                  = "password1"
	CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT = "1"
)

func newClickHouse(opts ...OptionsFunc) (*sql.DB, func(), error) {
	option := applyOptions(opts)
	var db *sql.DB
	ready := func(ctx context.Context, container *dockermanage.Container) error {
		// Port 8123 is used for HTTP, but we're using the TCP protocol endpoint (port 9000).
		// Ref: https://clickhouse.com/docs/en/interfaces/http/
		// Ref: https://clickhouse.com/docs/en/interfaces/tcp/
		address, err := container.Addr(9000)
		if err != nil {
			return err
		}
		candidate := clickHouseOpenDB(address, nil, option.debug)
		if err := candidate.PingContext(ctx); err != nil {
			return errors.Join(err, candidate.Close())
		}
		db = candidate
		return nil
	}
	_, cleanup, err := startContainer(option, CLICKHOUSE_IMAGE+":"+CLICKHOUSE_VERSION, readyTimeout, ready,
		portOption(option, 9000),
		dockermanage.WithEnv("CLICKHOUSE_DB", CLICKHOUSE_DB),
		dockermanage.WithEnv("CLICKHOUSE_USER", CLICKHOUSE_USER),
		dockermanage.WithEnv("CLICKHOUSE_PASSWORD", CLICKHOUSE_PASSWORD),
		dockermanage.WithEnv("CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT", CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT),
	)
	if err != nil {
		return nil, nil, err
	}
	return db, cleanup, nil
}

func clickHouseOpenDB(address string, tlsConfig *tls.Config, debug bool) *sql.DB {
	db := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{address},
		Auth: clickhouse.Auth{
			Database: CLICKHOUSE_DB,
			Username: CLICKHOUSE_USER,
			Password: CLICKHOUSE_PASSWORD,
		},
		TLS: tlsConfig,
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 5 * time.Second,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		Debug: debug,
	})
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(time.Hour)
	return db
}
