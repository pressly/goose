package database_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3/database"
	"github.com/pressly/goose/v3/internal/check"
	"go.uber.org/multierr"
	"modernc.org/sqlite"
)

// The goal of this test is to verify the database store package works as expected. This test is not
// meant to be exhaustive or test every possible database dialect. It is meant to verify the Store
// interface works against a real database.

func TestDialectStore(t *testing.T) {
	t.Parallel()
	t.Run("invalid", func(t *testing.T) {
		// Test empty table name.
		_, err := database.NewStore(database.DialectSQLite3, "")
		check.HasError(t, err)
		// Test unknown dialect.
		_, err = database.NewStore("unknown-dialect", "foo")
		check.HasError(t, err)
		// Test empty dialect.
		_, err = database.NewStore("", "foo")
		check.HasError(t, err)
	})
	// Test generic behavior.
	t.Run("sqlite3", func(t *testing.T) {
		db, err := sql.Open("sqlite", ":memory:")
		check.NoError(t, err)
		testStore(context.Background(), t, database.DialectSQLite3, db, func(t *testing.T, err error) {
			var sqliteErr *sqlite.Error
			ok := errors.As(err, &sqliteErr)
			check.Bool(t, ok, true)
			check.Equal(t, sqliteErr.Code(), 1) // Generic error (SQLITE_ERROR)
			check.Contains(t, sqliteErr.Error(), "table test_goose_db_version already exists")
		})
	})
	t.Run("ListMigrations", func(t *testing.T) {
		dir := t.TempDir()
		db, err := sql.Open("sqlite", filepath.Join(dir, "sql_embed.db"))
		check.NoError(t, err)
		store, err := database.NewStore(database.DialectSQLite3, "foo")
		check.NoError(t, err)
		err = store.CreateVersionTable(context.Background(), db)
		check.NoError(t, err)
		insert := func(db *sql.DB, version int64) error {
			return store.Insert(context.Background(), db, database.InsertRequest{Version: version})
		}
		check.NoError(t, insert(db, 1))
		check.NoError(t, insert(db, 3))
		check.NoError(t, insert(db, 2))
		res, err := store.ListMigrations(context.Background(), db)
		check.NoError(t, err)
		check.Number(t, len(res), 3)
		// Check versions are in descending order: [2, 3, 1]
		check.Number(t, res[0].Version, 2)
		check.Number(t, res[1].Version, 3)
		check.Number(t, res[2].Version, 1)
	})
}

// testStore tests various store operations.
//
// If alreadyExists is not nil, it will be used to assert the error returned by CreateVersionTable
// when the version table already exists.
func testStore(
	ctx context.Context,
	t *testing.T,
	d database.Dialect,
	db *sql.DB,
	alreadyExists func(t *testing.T, err error),
) {
	const (
		tablename = "test_goose_db_version"
	)
	store, err := database.NewStore(d, tablename)
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
	// Get the latest version. There should be none.
	_, err = store.GetLatestVersion(ctx, db)
	check.IsError(t, err, database.ErrVersionNotFound)

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
			err := store.Insert(ctx, conn, database.InsertRequest{Version: int64(i)})
			check.NoError(t, err)
			latest, err := store.GetLatestVersion(ctx, conn)
			check.NoError(t, err)
			check.Number(t, latest, int64(i))
			return nil
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
			err := store.Delete(ctx, conn, int64(i))
			check.NoError(t, err)
			latest, err := store.GetLatestVersion(ctx, conn)
			check.NoError(t, err)
			check.Number(t, latest, int64(i-1))
			return nil
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

	// Delete remaining migrations one by one and use all 3 connection types:

	// 1. *sql.Tx
	err = runTx(ctx, db, func(tx *sql.Tx) error {
		err := store.Delete(ctx, tx, 2)
		check.NoError(t, err)
		latest, err := store.GetLatestVersion(ctx, tx)
		check.NoError(t, err)
		check.Number(t, latest, 1)
		return nil
	})
	check.NoError(t, err)
	// 2. *sql.Conn
	err = runConn(ctx, db, func(conn *sql.Conn) error {
		err := store.Delete(ctx, conn, 1)
		check.NoError(t, err)
		latest, err := store.GetLatestVersion(ctx, conn)
		check.NoError(t, err)
		check.Number(t, latest, 0)
		return nil
	})
	check.NoError(t, err)
	// 3. *sql.DB
	err = store.Delete(ctx, db, 0)
	check.NoError(t, err)
	_, err = store.GetLatestVersion(ctx, db)
	check.IsError(t, err, database.ErrVersionNotFound)

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
		check.Bool(t, errors.Is(err, database.ErrVersionNotFound), true)
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
