package sqladapter_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
	"github.com/pressly/goose/v3/internal/sqladapter"
	"github.com/pressly/goose/v3/internal/testdb"
	"go.uber.org/multierr"
	"modernc.org/sqlite"
	_ "modernc.org/sqlite"
)

// The goal of this test is to verify the sqladapter package works as expected. This test is not
// meant to be exhaustive or test every possible database dialect. It is meant to verify the Store
// interface works against a real database.

func TestStore(t *testing.T) {
	t.Parallel()
	t.Run("postgres", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip long-running test")
		}
		// Test postgres specific behavior.
		db, cleanup, err := testdb.NewPostgres()
		check.NoError(t, err)
		t.Cleanup(cleanup)
		testStore(context.Background(), t, goose.DialectPostgres, db, func(t *testing.T, err error) {
			var pgErr *pgconn.PgError
			ok := errors.As(err, &pgErr)
			check.Bool(t, ok, true)
			check.Equal(t, pgErr.Code, "42P07") // duplicate_table
		})
	})
	// Test generic behavior.
	t.Run("sqlite3", func(t *testing.T) {
		dir := t.TempDir()
		db, err := sql.Open("sqlite", filepath.Join(dir, "sql_embed.db"))
		check.NoError(t, err)
		testStore(context.Background(), t, goose.DialectSQLite3, db, func(t *testing.T, err error) {
			var sqliteErr *sqlite.Error
			ok := errors.As(err, &sqliteErr)
			check.Bool(t, ok, true)
			check.Equal(t, sqliteErr.Code(), 1) // Generic error (SQLITE_ERROR)
			check.Contains(t, sqliteErr.Error(), "table test_goose_db_version already exists")
		})
	})
}

// testStore tests various store operations.
//
// If alreadyExists is not nil, it will be used to assert the error returned by CreateVersionTable
// when the version table already exists.
func testStore(ctx context.Context, t *testing.T, dialect goose.Dialect, db *sql.DB, alreadyExists func(t *testing.T, err error)) {
	const (
		tablename = "test_goose_db_version"
	)
	store, err := sqladapter.NewStore(string(dialect), tablename)
	check.NoError(t, err)
	// Create the version table.
	err = runTx(ctx, db, func(tx *sql.Tx) error {
		return store.CreateVersionTable(ctx, tx)
	})
	check.NoError(t, err)
	// Create the version table again. This should fail.
	err = runTx(ctx, db, func(tx *sql.Tx) error {
		return store.CreateVersionTable(ctx, tx)
	})
	check.HasError(t, err)
	if alreadyExists != nil {
		alreadyExists(t, err)
	}

	// List migrations. There should be none.
	err = runConn(ctx, db, func(conn *sql.Conn) error {
		res, err := store.ListMigrations(ctx, conn)
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
		res, err := store.ListMigrations(ctx, conn)
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
		res, err := store.ListMigrations(ctx, conn)
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
			res, err := store.GetMigration(ctx, conn, int64(i))
			check.NoError(t, err)
			check.Equal(t, res.IsApplied, true)
			check.Equal(t, res.Timestamp.IsZero(), false)
			return nil
		})
		check.NoError(t, err)
	}

	// Delete remaining migrations one by one and use all 3 connection types: 1. *sql.Tx
	err = runTx(ctx, db, func(tx *sql.Tx) error {
		return store.InsertOrDelete(ctx, tx, false, 2)
	})
	check.NoError(t, err)
	// 2. *sql.Conn
	err = runConn(ctx, db, func(conn *sql.Conn) error {
		return store.InsertOrDelete(ctx, conn, false, 1)
	})
	check.NoError(t, err)
	// 3. *sql.DB
	err = store.InsertOrDelete(ctx, db, false, 0)
	check.NoError(t, err)

	// List migrations. There should be none.
	err = runConn(ctx, db, func(conn *sql.Conn) error {
		res, err := store.ListMigrations(ctx, conn)
		check.NoError(t, err)
		check.Number(t, len(res), 0)
		return nil
	})
	check.NoError(t, err)

	// Try to get a migration that does not exist.
	err = runConn(ctx, db, func(conn *sql.Conn) error {
		_, err := store.GetMigration(ctx, conn, 0)
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
			retErr = multierr.Append(retErr, tx.Rollback())
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
			retErr = multierr.Append(retErr, conn.Close())
		}
	}()
	if err := fn(conn); err != nil {
		return err
	}
	return conn.Close()
}
