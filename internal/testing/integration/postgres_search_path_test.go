package integration

import (
	"context"
	"database/sql"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/pressly/goose/v3/internal/testing/testdb"
	"github.com/stretchr/testify/require"
)

// TestPostgresVersionTableOnSearchPath is a regression test for
// https://github.com/pressly/goose/issues/1017. An unqualified version table that lives in a schema
// further down the search_path (not current_schema()) must still be found, because that is the
// table every other goose query resolves to.
func TestPostgresVersionTableOnSearchPath(t *testing.T) {
	t.Parallel()

	db, cleanup, err := testdb.NewPostgres()
	require.NoError(t, err)
	t.Cleanup(cleanup)

	testPostgresVersionTableOnSearchPath(t, db)
}

func testPostgresVersionTableOnSearchPath(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()

	// Pin a single connection so the session-level search_path applies to every query.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	fsys := fstest.MapFS{
		"00001_a.sql": {Data: []byte("-- +goose Up\nCREATE TABLE a (id int);\n-- +goose Down\nDROP TABLE a;\n")},
		"00002_b.sql": {Data: []byte("-- +goose Up\nCREATE TABLE b (id int);\n-- +goose Down\nDROP TABLE b;\n")},
	}

	// Apply all migrations with the version table created in public.
	_, err := db.ExecContext(ctx, "SET search_path TO public")
	require.NoError(t, err)
	p, err := goose.NewProvider(database.DialectPostgres, db, fsys)
	require.NoError(t, err)
	res, err := p.Up(ctx)
	require.NoError(t, err)
	require.Len(t, res, 2)

	// A schema now comes first on the search_path (e.g., a "$user" schema created later), so
	// current_schema() is no longer where the version table lives, but the table is still visible.
	_, err = db.ExecContext(ctx, "CREATE SCHEMA app")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "SET search_path TO app, public")
	require.NoError(t, err)

	p, err = goose.NewProvider(database.DialectPostgres, db, fsys)
	require.NoError(t, err)
	res, err = p.Up(ctx)
	require.NoError(t, err)
	require.Empty(t, res, "migrations already applied must not run again")
	version, err := p.GetDBVersion(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 2, version)

	var exists bool
	err = db.QueryRowContext(ctx, "SELECT to_regclass('app.goose_db_version') IS NOT NULL").Scan(&exists)
	require.NoError(t, err)
	require.False(t, exists, "a second version table must not be created in the app schema")
}
