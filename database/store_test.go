package database_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3/database"
	"github.com/stretchr/testify/require"
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
		require.Error(t, err)
		// Test unknown dialect.
		_, err = database.NewStore("unknown-dialect", "foo")
		require.Error(t, err)
		// Test empty dialect.
		_, err = database.NewStore("", "foo")
		require.Error(t, err)
	})
	// Test generic behavior.
	t.Run("sqlite3", func(t *testing.T) {
		db, err := sql.Open("sqlite", ":memory:")
		require.NoError(t, err)
		testStore(context.Background(), t, database.DialectSQLite3, db, func(t *testing.T, err error) {
			t.Helper()
			var sqliteErr *sqlite.Error
			ok := errors.As(err, &sqliteErr)
			require.True(t, ok)
			require.Equal(t, 1, sqliteErr.Code()) // Generic error (SQLITE_ERROR)
			require.Contains(t, sqliteErr.Error(), "table test_goose_db_version already exists")
		})
	})
	t.Run("ListMigrations", func(t *testing.T) {
		dir := t.TempDir()
		db, err := sql.Open("sqlite", filepath.Join(dir, "sql_embed.db"))
		require.NoError(t, err)
		store, err := database.NewStore(database.DialectSQLite3, "foo")
		require.NoError(t, err)
		err = store.CreateVersionTable(context.Background(), db)
		require.NoError(t, err)
		insert := func(db *sql.DB, version int64) error {
			return store.Insert(context.Background(), db, database.InsertRequest{Version: version})
		}
		require.NoError(t, insert(db, 1))
		require.NoError(t, insert(db, 3))
		require.NoError(t, insert(db, 2))
		res, err := store.ListMigrations(context.Background(), db)
		require.NoError(t, err)
		require.Len(t, res, 3)
		// Check versions are in descending order: [2, 3, 1]
		require.EqualValues(t, 2, res[0].Version)
		require.EqualValues(t, 3, res[1].Version)
		require.EqualValues(t, 1, res[2].Version)
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
	require.NoError(t, err)
	// Create the version table.
	err = runTx(ctx, db, func(tx *sql.Tx) error {
		return store.CreateVersionTable(ctx, tx)
	})
	require.NoError(t, err)
	// Create the version table again. This should fail.
	err = runTx(ctx, db, func(tx *sql.Tx) error {
		return store.CreateVersionTable(ctx, tx)
	})
	require.Error(t, err)
	if alreadyExists != nil {
		alreadyExists(t, err)
	}
	// Get the latest version. There should be none.
	_, err = store.GetLatestVersion(ctx, db)
	require.ErrorIs(t, err, database.ErrVersionNotFound)

	// List migrations. There should be none.
	err = runConn(ctx, db, func(conn *sql.Conn) error {
		res, err := store.ListMigrations(ctx, conn)
		require.NoError(t, err)
		require.Empty(t, res, 0)
		return nil
	})
	require.NoError(t, err)

	// Insert 5 migrations in addition to the zero migration.
	for i := 0; i < 6; i++ {
		err = runConn(ctx, db, func(conn *sql.Conn) error {
			err := store.Insert(ctx, conn, database.InsertRequest{Version: int64(i)})
			require.NoError(t, err)
			latest, err := store.GetLatestVersion(ctx, conn)
			require.NoError(t, err)
			require.Equal(t, latest, int64(i))
			return nil
		})
		require.NoError(t, err)
	}

	// List migrations. There should be 6.
	err = runConn(ctx, db, func(conn *sql.Conn) error {
		res, err := store.ListMigrations(ctx, conn)
		require.NoError(t, err)
		require.Len(t, res, 6)
		// Check versions are in descending order.
		for i := 0; i < 6; i++ {
			require.EqualValues(t, res[i].Version, 5-i)
		}
		return nil
	})
	require.NoError(t, err)

	// Delete 3 migrations backwards
	for i := 5; i >= 3; i-- {
		err = runConn(ctx, db, func(conn *sql.Conn) error {
			err := store.Delete(ctx, conn, int64(i))
			require.NoError(t, err)
			latest, err := store.GetLatestVersion(ctx, conn)
			require.NoError(t, err)
			require.Equal(t, latest, int64(i-1))
			return nil
		})
		require.NoError(t, err)
	}

	// List migrations. There should be 3.
	err = runConn(ctx, db, func(conn *sql.Conn) error {
		res, err := store.ListMigrations(ctx, conn)
		require.NoError(t, err)
		require.Len(t, res, 3)
		// Check that the remaining versions are in descending order.
		for i := 0; i < 3; i++ {
			require.EqualValues(t, res[i].Version, 2-i)
		}
		return nil
	})
	require.NoError(t, err)

	// Get remaining migrations one by one.
	for i := 0; i < 3; i++ {
		err = runConn(ctx, db, func(conn *sql.Conn) error {
			res, err := store.GetMigration(ctx, conn, int64(i))
			require.NoError(t, err)
			require.True(t, res.IsApplied)
			require.False(t, res.Timestamp.IsZero())
			return nil
		})
		require.NoError(t, err)
	}

	// Delete remaining migrations one by one and use all 3 connection types:

	// 1. *sql.Tx
	err = runTx(ctx, db, func(tx *sql.Tx) error {
		err := store.Delete(ctx, tx, 2)
		require.NoError(t, err)
		latest, err := store.GetLatestVersion(ctx, tx)
		require.NoError(t, err)
		require.EqualValues(t, 1, latest)
		return nil
	})
	require.NoError(t, err)
	// 2. *sql.Conn
	err = runConn(ctx, db, func(conn *sql.Conn) error {
		err := store.Delete(ctx, conn, 1)
		require.NoError(t, err)
		latest, err := store.GetLatestVersion(ctx, conn)
		require.NoError(t, err)
		require.EqualValues(t, 0, latest)
		return nil
	})
	require.NoError(t, err)
	// 3. *sql.DB
	err = store.Delete(ctx, db, 0)
	require.NoError(t, err)
	_, err = store.GetLatestVersion(ctx, db)
	require.ErrorIs(t, err, database.ErrVersionNotFound)

	// List migrations. There should be none.
	err = runConn(ctx, db, func(conn *sql.Conn) error {
		res, err := store.ListMigrations(ctx, conn)
		require.NoError(t, err)
		require.Empty(t, res)
		return nil
	})
	require.NoError(t, err)

	// Try to get a migration that does not exist.
	err = runConn(ctx, db, func(conn *sql.Conn) error {
		_, err := store.GetMigration(ctx, conn, 0)
		require.Error(t, err)
		require.ErrorIs(t, err, database.ErrVersionNotFound)
		return nil
	})
	require.NoError(t, err)
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
