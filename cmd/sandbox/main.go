package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/pressly/goose/v3/internal/testdb"
	"github.com/ydb-platform/ydb-go-sdk/v3"
)

func getFromE2E() {
	db, cleanup, err := testdb.NewYdb()
	if err != nil {
		log.Fatal("newYdb:", err)
	}
	defer cleanup()

	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(time.Hour)

	query := `
	CREATE TABLE owners (
		owner_id Uint64,
		owner_name Utf8,
		owner_type Utf8,
		PRIMARY KEY (owner_id)
	);`
	_, err = db.ExecContext(context.Background(), query)
	if err != nil {
		log.Fatal("ping:", err)
	}
}

func getFromManual() {
	nativeDriver, err := ydb.Open(context.Background(), "grpc://localhost:2136/local")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err != nil {
			nativeDriver.Close(context.Background())
		}
	}()
	connector, err := ydb.Connector(nativeDriver,
		ydb.WithDefaultQueryMode(ydb.ScriptingQueryMode),
		ydb.WithFakeTx(ydb.ScriptingQueryMode),
		ydb.WithAutoDeclare(),
		ydb.WithNumericArgs(),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err != nil {
			connector.Close()
		}
	}()

	db := sql.OpenDB(connector)
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(time.Hour)

	query := `
	CREATE TABLE owners (
		owner_id Uint64,
		owner_name Utf8,
		owner_type Utf8,
		PRIMARY KEY (owner_id)
	);`
	_, err = db.ExecContext(context.Background(), query)
	if err != nil {
		log.Fatal("ping:", err)
	}
}

func main() {
	//getFromE2E()
	getFromManual()
}
