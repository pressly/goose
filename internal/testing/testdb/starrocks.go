package testdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pressly/dockermanage"
)

const (
	// https://hub.docker.com/r/starrocks/allin1-ubuntu
	STARROCKS_IMAGE   = "starrocks/allin1-ubuntu"
	STARROCKS_VERSION = "3.5.11"

	STARROCKS_USER    = "root"
	STARROCKS_INIT_DB = "migrations"
)

func newStarrocks(opts ...OptionsFunc) (*sql.DB, func(), error) {
	option := applyOptions(opts)
	var db *sql.DB
	ready := func(ctx context.Context, container *dockermanage.Container) error {
		addr, err := container.Addr(9030)
		if err != nil {
			return err
		}
		dsn := fmt.Sprintf("%s:%s@(%s)/%s?parseTime=true&interpolateParams=true",
			STARROCKS_USER,
			"",
			addr,
			"",
		)
		candidate, err := sql.Open("mysql", dsn)
		if err != nil {
			return err
		}
		if err := starrocksInit(ctx, candidate); err != nil {
			return errors.Join(err, candidate.Close())
		}
		db = candidate
		return nil
	}
	// The frontend can accept connections before a backend is available, and the backend takes
	// longer to start than the other databases.
	_, cleanup, err := startContainer(option, STARROCKS_IMAGE+":"+STARROCKS_VERSION, 2*time.Minute, ready,
		portOption(option, 9030),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("could not connect to docker database: %v", err)
	}
	return db, cleanup, nil
}

// starrocksInit creates the initial database and checks that a backend is alive. Creating a table
// requires at least one alive backend.
func starrocksInit(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+STARROCKS_INIT_DB); err != nil {
		return fmt.Errorf("could not create initial database: %v", err)
	}
	if _, err := db.ExecContext(ctx, "USE "+STARROCKS_INIT_DB); err != nil {
		return fmt.Errorf("could not set default initial database: %v", err)
	}
	return checkStarrocksBackend(db)
}

func checkStarrocksBackend(db *sql.DB) error {
	rows, err := db.Query("SHOW BACKENDS")
	if err != nil {
		return fmt.Errorf("could not query StarRocks backends: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("could not get StarRocks backend columns: %v", err)
	}
	aliveIndex := -1
	for i, column := range columns {
		if strings.EqualFold(column, "Alive") {
			aliveIndex = i
			break
		}
	}
	if aliveIndex == -1 {
		return fmt.Errorf("StarRocks backend status does not include an Alive column")
	}

	values := make([]sql.RawBytes, len(columns))
	dest := make([]any, len(columns))
	for i := range values {
		dest[i] = &values[i]
	}
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return fmt.Errorf("could not scan StarRocks backend status: %v", err)
		}
		if strings.EqualFold(string(values[aliveIndex]), "true") {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("could not read StarRocks backend status: %v", err)
	}
	return fmt.Errorf("no alive StarRocks backends")
}
