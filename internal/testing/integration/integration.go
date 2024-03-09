package integration

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/stretchr/testify/require"
)

func collectMigrations(t *testing.T, dir string) []string {
	t.Helper()

	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	all := make([]string, 0, len(files))
	for _, f := range files {
		require.False(t, f.IsDir())
		all = append(all, f.Name())
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
	// run all up migrations
	results, err := p.Up(ctx)
	require.NoError(t, err)
	require.Equal(t, len(wantFiles), len(results), "number of migrations applied")
	for i, r := range results {
		require.Equal(t, wantFiles[i], r.Source.Path, "migration file")
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
		require.Equal(t, wantFiles[i], result.Source.Path, "migration file")
	}
	// check the current version
	currentVersion, err = p.GetDBVersion(ctx)
	require.NoError(t, err)
	require.Equal(t, len(wantFiles), int(currentVersion), "current version")
}
