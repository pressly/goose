package testdb

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pressly/dockermanage"
)

const (
	// https://hub.docker.com/_/mariadb
	MARIADB_IMAGE   = "mariadb"
	MARIADB_VERSION = "11"

	MARIADB_DB       = "testdb"
	MARIADB_USER     = "tester"
	MARIADB_PASSWORD = "password1"
)

func newMariaDB(opts ...OptionsFunc) (*sql.DB, func(), error) {
	option := applyOptions(opts)
	var db *sql.DB
	ready := func(ctx context.Context, container *dockermanage.Container) error {
		addr, err := container.Addr(3306)
		if err != nil {
			return err
		}
		// MySQL DSN: username:password@protocol(address)/dbname?param=value
		dsn := fmt.Sprintf("%s:%s@(%s)/%s?parseTime=true&multiStatements=true",
			MARIADB_USER,
			MARIADB_PASSWORD,
			addr,
			MARIADB_DB,
		)
		db, err = openAndPing(ctx, "mysql", dsn)
		return err
	}
	_, cleanup, err := startContainer(option, MARIADB_IMAGE+":"+MARIADB_VERSION, readyTimeout, ready,
		portOption(option, 3306),
		dockermanage.WithEnv("MARIADB_USER", MARIADB_USER),
		dockermanage.WithEnv("MARIADB_PASSWORD", MARIADB_PASSWORD),
		dockermanage.WithEnv("MARIADB_ROOT_PASSWORD", MARIADB_PASSWORD),
		dockermanage.WithEnv("MARIADB_DATABASE", MARIADB_DB),
	)
	if err != nil {
		return nil, nil, err
	}
	return db, cleanup, nil
}
