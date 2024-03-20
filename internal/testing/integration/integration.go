package integration

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/stretchr/testify/require"
)

type collected struct {
	fullpath string
	version  int64
}

func collectMigrations(t *testing.T, dir string) []collected {
	t.Helper()

	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	all := make([]collected, 0, len(files))
	for _, f := range files {
		require.False(t, f.IsDir())
		v, err := goose.NumericComponent(f.Name())
		require.NoError(t, err)
		all = append(all, collected{
			fullpath: filepath.Base(f.Name()),
			version:  v,
		})
	}
	return all
}

func testDatabase(t *testing.T, dialect database.Dialect, db *sql.DB, migrationsDir string) {
	t.Helper()

	ctx := context.Background()
	// collect all migration files from the testdata directory
	wantFiles := collectMigrations(t, migrationsDir)
	// initialize a new goose provider
	p, err := goose.NewProvider(dialect, db, os.DirFS(migrationsDir))
	require.NoError(t, err)
	require.Equal(t, len(wantFiles), len(p.ListSources()), "number of migrations")
	// run all up migrations
	results, err := p.Up(ctx)
	require.NoError(t, err)
	require.Equal(t, len(wantFiles), len(results), "number of migrations applied")
	for i, r := range results {
		require.Equal(t, wantFiles[i].fullpath, r.Source.Path, "migration file")
		require.Equal(t, wantFiles[i].version, r.Source.Version, "migration version")
	}
	// check the current version
	currentVersion, err := p.GetDBVersion(ctx)
	require.NoError(t, err)
	require.Equal(t, len(wantFiles), int(currentVersion), "current version")
	// run all down migrations
	results, err = p.DownTo(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, len(wantFiles), len(results), "number of migrations rolled back")
	// check the current version
	currentVersion, err = p.GetDBVersion(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, int(currentVersion), "current version")
	// run all up migrations one by one
	for i := range len(wantFiles) {
		result, err := p.UpByOne(ctx)
		require.NoError(t, err)
		if errors.Is(err, goose.ErrNoNextVersion) {
			break
		}
		require.Equal(t, wantFiles[i].fullpath, result.Source.Path, "migration file")
	}
	// check the current version
	currentVersion, err = p.GetDBVersion(ctx)
	require.NoError(t, err)
	require.Equal(t, len(wantFiles), int(currentVersion), "current version")
}
