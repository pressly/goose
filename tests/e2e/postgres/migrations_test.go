package postgres_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/pressly/goose/v4"
	"github.com/pressly/goose/v4/internal/check"
)

func TestUpDownAll(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tt := []struct {
		name         string
		maxOpenConns int
		maxIdleConns int
		useDefaults  bool
	}{
		// Single connection ensures goose is able to function correctly when multiple connections
		// are not available.
		{name: "single_conn", maxOpenConns: 1, maxIdleConns: 1},
		{name: "defaults", useDefaults: true},
	}
	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Start a new Docker container for each test case.
			te := newTestEnv(t, migrationsDir, nil)
			if !tc.useDefaults {
				te.db.SetMaxOpenConns(tc.maxOpenConns)
				te.db.SetMaxIdleConns(tc.maxIdleConns)
			}

			migrations := te.provider.ListMigrations()
			check.NumberNotZero(t, len(migrations))

			currentVersion, err := te.provider.GetDBVersion(ctx)
			check.NoError(t, err)
			check.Number(t, currentVersion, 0)

			{
				// Apply all up migrations
				upResult, err := te.provider.Up(ctx)
				check.NoError(t, err)
				check.Number(t, len(upResult), len(migrations))
				currentVersion, err := te.provider.GetDBVersion(ctx)
				check.NoError(t, err)
				check.Number(t, currentVersion, te.provider.GetLastVersion())
				// Validate the db migration version actually matches what goose claims it is
				gotVersion, err := getMaxVersionID(te.db, te.opt.TableName)
				check.NoError(t, err)
				check.Number(t, gotVersion, currentVersion)
				tables, err := getTableNames(te.db)
				check.NoError(t, err)
				if !reflect.DeepEqual(tables, knownTables) {
					t.Logf("got tables: %v", tables)
					t.Logf("known tables: %v", knownTables)
					t.Fatal("failed to match tables")
				}
			}
			{
				// Apply all down migrations
				downResult, err := te.provider.DownTo(ctx, 0)
				check.NoError(t, err)
				check.Number(t, len(downResult), len(migrations))
				gotVersion, err := getMaxVersionID(te.db, te.opt.TableName)
				check.NoError(t, err)
				check.Number(t, gotVersion, 0)
				// Should only be left with a single table, the default goose table
				tables, err := getTableNames(te.db)
				check.NoError(t, err)
				knownTables := []string{te.opt.TableName}
				if !reflect.DeepEqual(tables, knownTables) {
					t.Logf("got tables: %v", tables)
					t.Logf("known tables: %v", knownTables)
					t.Fatal("failed to match tables")
				}
			}
		})
	}
}

func TestMigrateUpTo(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	const (
		upToVersion int64 = 2
	)

	te := newTestEnv(t, migrationsDir, nil)
	check.NumberNotZero(t, len(te.provider.ListMigrations()))

	results, err := te.provider.UpTo(ctx, upToVersion)
	check.NoError(t, err)
	check.Number(t, len(results), upToVersion)
	// Fetch the goose version from DB
	currentVersion, err := te.provider.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, currentVersion, upToVersion)
	// Validate the version actually matches what goose claims it is
	gotVersion, err := getMaxVersionID(te.db, te.opt.TableName)
	check.NoError(t, err)
	check.Number(t, gotVersion, upToVersion)
}

func TestMigrateUpByOneWithRedo(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	te := newTestEnv(t, migrationsDir, nil)
	migrations := te.provider.ListMigrations()
	check.NumberNotZero(t, len(migrations))
	maxVersion := te.provider.GetLastVersion()

	for i := 0; i < len(migrations); i++ {
		originalUpResult, err := te.provider.UpByOne(ctx)
		check.NoError(t, err)
		// Redo the previous Up migration and re-apply it.
		result, err := te.provider.Redo(ctx)
		check.NoError(t, err)
		check.Number(t, len(result), 2)
		check.Equal(t, result[0].Version, originalUpResult.Version)
		check.Equal(t, result[1].Version, originalUpResult.Version)

		currentVersion, err := te.provider.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, migrations[i].Version)
	}
	// Once everything is tested the version should match the highest testdata version
	currentVersion, err := te.provider.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, currentVersion, maxVersion)
}

func TestMigrateUpByOne(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	te := newTestEnv(t, migrationsDir, nil)
	migrations := te.provider.ListMigrations()
	check.NumberNotZero(t, len(migrations))
	maxVersion := te.provider.GetLastVersion()

	// Apply all migrations one-by-one.
	var counter int
	for {
		result, err := te.provider.UpByOne(ctx)
		counter++
		if counter > len(migrations) {
			if !errors.Is(err, goose.ErrNoNextVersion) {
				t.Fatalf("incorrect error: got:%v want:%v", err, goose.ErrNoNextVersion)
			}
			break
		}
		check.NoError(t, err)
		check.Number(t, result.Version, counter)
	}
	// Once everything is tested the version should match the highest testdata version
	currentVersion, err := te.provider.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, currentVersion, maxVersion)
}
