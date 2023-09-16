package sqladapter_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
	"github.com/pressly/goose/v3/internal/sqladapter"
	"github.com/pressly/goose/v3/internal/testdb"
)

// The goal of this test is to verify the sqladapter package works as expected. This test is not
// meant to be exhaustive or test every possible database dialect. It is meant to verify that the
// Store interface works against a real database.

func TestStore_Postgres(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skip long-running test")
	}
	ctx := context.Background()
	const (
		tablename = "test_goose_db_version"
	)
	db, cleanup, err := testdb.NewPostgres()
	check.NoError(t, err)
	t.Cleanup(cleanup)
	store, err := sqladapter.NewStore(string(goose.DialectPostgres), tablename)
	check.NoError(t, err)
	// Create the version table.
	err = runTx(ctx, db, func(tx *sql.Tx) error {
		return store.CreateVersionTable(ctx, tx, tablename)
	})
	// Create the version table again. This should fail.
	check.NoError(t, err)
	err = runTx(ctx, db, func(tx *sql.Tx) error {
		return store.CreateVersionTable(ctx, tx, tablename)
	})
	check.HasError(t, err)
	var pgErr *pgconn.PgError
	ok := errors.As(err, &pgErr)
	check.Bool(t, ok, true)
	check.Equal(t, pgErr.Code, "42P07") // duplicate_table
	// List migrations. There should be none.
	err = runConn(ctx, db, func(conn *sql.Conn) error {
		res, err := store.ListMigrationsConn(ctx, conn)
		check.NoError(t, err)
		check.Number(t, len(res), 0)
		return nil
	})
	check.NoError(t, err)
	// Insert 5 migrations in addition to the zero migration.
	for i := 0; i < 6; i++ {
		err = runConn(ctx, db, func(conn *sql.Conn) error {
			return store.InsertOrDelete(ctx, conn, true, int64(i))
		})
		check.NoError(t, err)
	}
	// List migrations. There should be 6.
	err = runConn(ctx, db, func(conn *sql.Conn) error {
		res, err := store.ListMigrationsConn(ctx, conn)
		check.NoError(t, err)
		check.Number(t, len(res), 6)
		// Check versions are in descending order.
		for i := 0; i < 6; i++ {
			check.Number(t, res[i].Version, 5-i)
		}
		return nil
	})
	check.NoError(t, err)
	// Delete 3 migrations backwards
	for i := 5; i >= 3; i-- {
		err = runConn(ctx, db, func(conn *sql.Conn) error {
			return store.InsertOrDelete(ctx, conn, false, int64(i))
		})
		check.NoError(t, err)
	}
	// List migrations. There should be 3.
	err = runConn(ctx, db, func(conn *sql.Conn) error {
		res, err := store.ListMigrationsConn(ctx, conn)
		check.NoError(t, err)
		check.Number(t, len(res), 3)
		// Check that the remaining versions are in descending order.
		for i := 0; i < 3; i++ {
			check.Number(t, res[i].Version, 2-i)
		}
		return nil
	})
	check.NoError(t, err)
	// Get remaining migrations one by one.
	for i := 0; i < 3; i++ {
		err = runConn(ctx, db, func(conn *sql.Conn) error {
			res, err := store.GetMigrationConn(ctx, conn, int64(i))
			check.NoError(t, err)
			check.Equal(t, res.IsApplied, true)
			check.Equal(t, res.Timestamp.IsZero(), false)
			return nil
		})
		check.NoError(t, err)
	}
	// Delete remaining migrations one by one and use all 3 connection types:
	// *sql.DB
	// *sql.Tx
	// *sql.Conn.
	err = runTx(ctx, db, func(tx *sql.Tx) error {
		return store.InsertOrDelete(ctx, tx, false, 2) // *sql.Tx
	})
	check.NoError(t, err)
	err = runConn(ctx, db, func(conn *sql.Conn) error {
		return store.InsertOrDelete(ctx, conn, false, 1) // *sql.Conn
	})
	check.NoError(t, err)
	err = store.InsertOrDelete(ctx, db, false, 0) // *sql.DB
	check.NoError(t, err)
	// List migrations. There should be none.
	err = runConn(ctx, db, func(conn *sql.Conn) error {
		res, err := store.ListMigrationsConn(ctx, conn)
		check.NoError(t, err)
		check.Number(t, len(res), 0)
		return nil
	})
	check.NoError(t, err)
	// Try to get a migration that does not exist.
	err = runConn(ctx, db, func(conn *sql.Conn) error {
		_, err := store.GetMigrationConn(ctx, conn, 0)
		check.HasError(t, err)
		check.Bool(t, errors.Is(err, sql.ErrNoRows), true)
		return nil
	})
	check.NoError(t, err)
}

func runTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) (retErr error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, tx.Rollback())
		}
	}()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func runConn(ctx context.Context, db *sql.DB, fn func(*sql.Conn) error) (retErr error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, conn.Close())
		}
	}()
	if err := fn(conn); err != nil {
		return err
	}
	return conn.Close()
}
