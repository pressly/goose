//go:build chdb

package goose_test

// This test exercises the real chdb-go database/sql driver. It is build-tagged because chdb-go
// loads libchdb during package initialization; install libchdb before running with -tags chdb.

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/stretchr/testify/require"

	_ "github.com/chdb-io/chdb-go/chdb/driver"
)

func TestProviderChDBIsolateDDLDisablesTransactions(t *testing.T) {
	fsys := fstest.MapFS{
		"00001_create_chdb_probe.sql": {
			Data: []byte(`
-- +goose Up
CREATE TABLE chdb_probe (
	id UInt64
) ENGINE = MergeTree()
ORDER BY id;
INSERT INTO chdb_probe (id) VALUES (1);

-- +goose Down
DROP TABLE chdb_probe;
`),
		},
	}

	t.Run("without isolate ddl", func(t *testing.T) {
		db := openChDB(t)
		p, err := goose.NewProvider(
			goose.DialectClickHouse,
			db,
			fsys,
			goose.WithStorePlaceholderFormat(database.PlaceholderQuestion),
		)
		require.NoError(t, err)

		_, err = p.Up(context.Background())
		require.Error(t, err)
		// failed to initialize: create version table: does not support Transcation
		require.Contains(t, err.Error(), "does not support")
	})

	t.Run("clickhouse dialect with isolate ddl", func(t *testing.T) {
		db := openChDB(t)
		p, err := goose.NewProvider(goose.DialectClickHouse, db, fsys, goose.WithIsolateDDL(true))
		require.NoError(t, err)

		_, err = p.Up(context.Background())
		require.Error(t, err)
		// failed to initialize: insert zero version: failed to insert version 0: Code: 47.
		// DB::Exception: Unknown expression identifier `$1` in scope `$1`: While executing
		// ValuesBlockInputFormat: data for INSERT was parsed from query. (UNKNOWN_IDENTIFIER)
		// (version 26.1.2.1). (UNKNOWN_IDENTIFIER)
		require.Contains(t, err.Error(), "Unknown expression identifier `$1`")
	})

	t.Run("with isolate ddl", func(t *testing.T) {
		db := openChDB(t)
		p, err := goose.NewProvider(
			goose.DialectClickHouse,
			db,
			fsys,
			goose.WithIsolateDDL(true),
			goose.WithStorePlaceholderFormat(database.PlaceholderQuestion),
		)
		require.NoError(t, err)

		res, err := p.Up(context.Background())
		require.NoError(t, err)
		require.Len(t, res, 1)

		var count uint64
		require.NoError(t, db.QueryRowContext(context.Background(), "SELECT count() FROM chdb_probe").Scan(&count))
		require.EqualValues(t, 1, count)
	})
}

func openChDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("chdb", fmt.Sprintf("session=%s", t.TempDir()))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	db.SetMaxOpenConns(1)
	require.NoError(t, db.PingContext(context.Background()))
	return db
}
