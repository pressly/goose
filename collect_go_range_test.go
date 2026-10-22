package goose_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestCollectMigrationsGoDirectoryRange(t *testing.T) {
	// The legacy registry is global and must not be used in parallel.
	goose.ResetGlobalMigrations()
	t.Cleanup(goose.ResetGlobalMigrations)
	goose.AddNamedMigrationContext("00010_local.go", nil, nil)
	goose.AddNamedMigrationContext("00020_other.go", nil, nil)

	for _, tc := range []struct {
		name    string
		files   []string
		current int64
		target  int64
		want    []int64
	}{
		{name: "local_below_range", files: []string{"00010_local.go", "00020_query.sql"}, current: 10, target: 20, want: []int64{20}},
		{name: "local_above_range", files: []string{"00010_local.go", "00020_query.sql"}, current: 20, target: 15, want: []int64{20}},
		{name: "empty_local_range", files: []string{"00010_local.go"}, current: 10, target: 20},
		{name: "local_in_range", files: []string{"00010_local.go"}, current: 0, target: 20, want: []int64{10}},
		{name: "manual_registry", current: 10, target: 20, want: []int64{20}},
		{name: "helper_only", files: []string{"helpers.go", "00010_local_test.go"}, current: 10, target: 20, want: []int64{20}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, file := range tc.files {
				require.NoError(t, os.WriteFile(filepath.Join(dir, file), nil, 0600))
			}
			var migrations goose.Migrations
			require.NotPanics(t, func() {
				var err error
				migrations, err = goose.CollectMigrations(dir, tc.current, tc.target)
				if len(tc.want) == 0 {
					require.ErrorIs(t, err, goose.ErrNoMigrationFiles)
				} else {
					require.NoError(t, err)
				}
			})
			var versions []int64
			for _, migration := range migrations {
				versions = append(versions, migration.Version)
			}
			require.Equal(t, tc.want, versions)
		})
	}
}

func TestUpToKeepsGoDirectoryScope(t *testing.T) {
	goose.ResetGlobalMigrations()
	t.Cleanup(goose.ResetGlobalMigrations)
	require.NoError(t, goose.SetDialect("sqlite3"))
	calls := 0
	goose.AddNamedMigration("00010_local.go", nil, nil)
	goose.AddNamedMigration("00020_other.go", func(*sql.Tx) error {
		calls++
		return nil
	}, nil)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "00010_local.go"), nil, 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "00020_query.sql"), []byte("-- +goose Up\nCREATE TABLE local_result (id INTEGER);\n-- +goose Down\nDROP TABLE local_result;\n"), 0600))
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	db.SetMaxOpenConns(1)
	require.NoError(t, goose.UpTo(db, dir, 10))
	require.NotPanics(t, func() { require.NoError(t, goose.UpTo(db, dir, 20)) })
	require.Zero(t, calls)
	version, err := goose.GetDBVersion(db)
	require.NoError(t, err)
	require.EqualValues(t, 20, version)
	_, err = db.Exec("INSERT INTO local_result (id) VALUES (1)")
	require.NoError(t, err)
}
