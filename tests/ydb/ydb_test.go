package ydb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
	"github.com/pressly/goose/v3/internal/testdb"
	"github.com/ydb-platform/ydb-go-sdk/v3"
)

func TestMigrateUpWithReset(t *testing.T) {
	db, extraNativeDriver, cleanup, err := testdb.NewYdbWithNative()
	check.NoError(t, err)
	t.Cleanup(cleanup)
	defer func() {
		_ = extraNativeDriver.Close(context.Background())
	}()
	migrationsDir := filepath.Join("testdata", "migrations")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = goose.SetDialect("ydb")
	check.NoError(t, err)

	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	check.NoError(t, err)
	check.NumberNotZero(t, len(migrations))
	err = goose.UpContext(ctx, db, migrationsDir)
	check.NoError(t, err)
	// Migrate all
	err = goose.UpContext(ctx, db, migrationsDir)
	check.NoError(t, err)
	currentVersion, err := goose.GetDBVersionContext(ctx, db)
	check.NoError(t, err)
	check.Number(t, currentVersion, migrations[len(migrations)-1].Version)
	// Validate the db migration version actually matches what goose claims it is
	gotVersion, err := getCurrentGooseVersion(ctx, db, goose.TableName())
	check.NoError(t, err)
	// incorrect database version
	check.Number(t, gotVersion, currentVersion)

	// Migrate everything down using Reset.
	err = goose.ResetContext(ctx, db, migrationsDir)
	check.NoError(t, err)
	currentVersion, err = goose.GetDBVersionContext(ctx, db)
	check.NoError(t, err)
	check.Number(t, currentVersion, 0)

}

func TestMigrateUpWithRedo(t *testing.T) {
	db, extraNativeDriver, cleanup, err := testdb.NewYdbWithNative()
	check.NoError(t, err)
	t.Cleanup(cleanup)
	defer func() {
		_ = extraNativeDriver.Close(context.Background())
	}()
	migrationsDir := filepath.Join("testdata", "migrations")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = goose.SetDialect("ydb")
	check.NoError(t, err)

	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	check.NoError(t, err)
	check.NumberNotZero(t, len(migrations))
	startingVersion, err := goose.EnsureDBVersionContext(ctx, db)
	check.NoError(t, err)
	check.Number(t, startingVersion, 0)
	// Migrate all
	for _, migration := range migrations {
		err = migration.Up(db)
		check.NoError(t, err)
		currentVersion, err := goose.GetDBVersionContext(ctx, db)
		check.NoError(t, err)
		check.Number(t, currentVersion, migration.Version)

		// Redo the previous Up migration and re-apply it.
		err = goose.RedoContext(ctx, db, migrationsDir)
		check.NoError(t, err)
		currentVersion, err = goose.GetDBVersionContext(ctx, db)
		check.NoError(t, err)
		check.Number(t, currentVersion, migration.Version)
	}
	// Once everything is tested the version should match the highest testdata version
	maxVersion := migrations[len(migrations)-1].Version
	currentVersion, err := goose.GetDBVersionContext(ctx, db)
	check.NoError(t, err)
	check.Number(t, currentVersion, maxVersion)
}

func TestMigrateUpTo(t *testing.T) {
	db, extraNativeDriver, cleanup, err := testdb.NewYdbWithNative()
	check.NoError(t, err)
	t.Cleanup(cleanup)
	defer func() {
		_ = extraNativeDriver.Close(context.Background())
	}()
	migrationsDir := filepath.Join("testdata", "migrations")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = goose.SetDialect("ydb")
	check.NoError(t, err)

	const (
		upToVersion int64 = 2
	)
	check.NoError(t, err)
	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	check.NoError(t, err)
	check.NumberNotZero(t, len(migrations))
	// Migrate up to the second migration
	err = goose.UpToContext(ctx, db, migrationsDir, upToVersion)
	check.NoError(t, err)
	// Fetch the goose version from DB
	currentVersion, err := goose.GetDBVersionContext(ctx, db)
	check.NoError(t, err)
	check.Number(t, currentVersion, upToVersion)
	// Validate the version actually matches what goose claims it is
	gotVersion, err := getCurrentGooseVersion(ctx, db, goose.TableName())
	check.NoError(t, err)
	check.Number(t, gotVersion, upToVersion) // incorrect database version
}

func TestMigrateUpByOne(t *testing.T) {
	db, extraNativeDriver, cleanup, err := testdb.NewYdbWithNative()
	check.NoError(t, err)
	t.Cleanup(cleanup)
	defer func() {
		_ = extraNativeDriver.Close(context.Background())
	}()
	migrationsDir := filepath.Join("testdata", "migrations")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = goose.SetDialect("ydb")
	check.NoError(t, err)

	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	check.NoError(t, err)
	check.NumberNotZero(t, len(migrations))
	// Apply all migrations one-by-one.
	var counter int
	for {
		err := goose.UpByOneContext(ctx, db, migrationsDir)
		counter++
		if counter > len(migrations) {
			if !errors.Is(err, goose.ErrNoNextVersion) {
				t.Fatalf("incorrect error: got:%v want:%v", err, goose.ErrNoNextVersion)
			}
			break
		}
		check.NoError(t, err)
	}
	currentVersion, err := goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, migrations[len(migrations)-1].Version)
	// Validate the db migration version actually matches what goose claims it is
	gotVersion, err := getCurrentGooseVersion(ctx, db, goose.TableName())
	check.NoError(t, err)
	check.Number(t, gotVersion, currentVersion) // incorrect database version
}

func TestMigrateFull(t *testing.T) {
	db, extraNativeDriver, cleanup, err := testdb.NewYdbWithNative()
	check.NoError(t, err)
	t.Cleanup(cleanup)
	defer func() {
		_ = extraNativeDriver.Close(context.Background())
	}()
	migrationsDir := filepath.Join("testdata", "migrations")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = goose.SetDialect("ydb")
	check.NoError(t, err)
	knownTables := map[string]struct{}{
		"goose_db_version": {},
		"issues":           {},
		"owners":           {},
		"repos":            {},
		"stargazers":       {},
	}

	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	check.NoError(t, err)
	check.NumberNotZero(t, len(migrations))
	// test retrieving invalid current goose migrations. goose cannot return a migration
	// if the supplied "current" version is non-existent or 0.
	_, err = migrations.Current(20160813)
	if !errors.Is(err, goose.ErrNoCurrentVersion) {
		t.Fatalf("incorrect error: got:%v want:%v", err, goose.ErrNoCurrentVersion)
	}
	_, err = migrations.Current(0)
	if !errors.Is(err, goose.ErrNoCurrentVersion) {
		t.Fatalf("incorrect error: got:%v want:%v", err, goose.ErrNoCurrentVersion)
	}
	// verify the first migration1. This should not change if more migrations are added
	// in the future.
	migration1, err := migrations.Current(1)
	check.NoError(t, err)
	check.Number(t, migration1.Version, 1)
	if migration1.Source != filepath.Join(migrationsDir, "00001_a.sql") {
		t.Fatalf("failed to get correct migration file:\ngot:%s\nwant:%s",
			migration1.Source,
			filepath.Join(migrationsDir, "00001_a.sql"),
		)
	}
	// expecting false for .sql migrations
	check.Bool(t, migration1.Registered, false)
	check.Number(t, migration1.Previous, -1)
	check.Number(t, migration1.Next, 2)

	{
		// Apply all up migrations
		err = goose.UpContext(ctx, db, migrationsDir)
		check.NoError(t, err)
		currentVersion, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, currentVersion, migrations[len(migrations)-1].Version)
		// Validate the db migration version actually matches what goose claims it is
		gotVersion, err := getCurrentGooseVersion(ctx, db, goose.TableName())
		check.NoError(t, err)
		check.Number(t, gotVersion, currentVersion) // incorrect database version
		tables, err := getAllTables(ctx, extraNativeDriver, testdb.YDB_DATABASE)
		check.NoError(t, err)
		if !reflect.DeepEqual(tables, knownTables) {
			t.Logf("got tables: %v", tables)
			t.Logf("known tables: %v", knownTables)
			t.Fatal("failed to match tables")
		}
	}
	{
		// Apply 1 down migration
		err := goose.Down(db, migrationsDir)
		check.NoError(t, err)
		gotVersion, err := getCurrentGooseVersion(ctx, db, goose.TableName())
		check.NoError(t, err)
		check.Number(t, gotVersion, migrations[len(migrations)-1].Version-1) // incorrect database version
	}
	{
		// Migrate all remaining migrations down. Should only be left with a single table:
		// the default goose table
		err := goose.DownTo(db, migrationsDir, 0)
		check.NoError(t, err)
		gotVersion, err := getCurrentGooseVersion(ctx, db, goose.TableName())
		check.NoError(t, err)
		check.Number(t, gotVersion, 0)
		tables, err := getAllTables(ctx, extraNativeDriver, testdb.YDB_DATABASE)
		check.NoError(t, err)
		knownTables := map[string]struct{}{
			"goose_db_version": {},
		}
		if !reflect.DeepEqual(tables, knownTables) {
			t.Logf("got tables: %v", tables)
			t.Logf("known tables: %v", knownTables)
			t.Fatal("failed to match tables")
		}
	}
}

func getCurrentGooseVersion(ctx context.Context, db *sql.DB, gooseTable string) (int64, error) {
	var gotVersion int64
	if err := db.QueryRowContext(
		ctx,
		fmt.Sprintf("SELECT MAX(version_id) FROM %s", gooseTable),
	).Scan(&gotVersion); err != nil {
		return 0, err
	}
	return gotVersion, nil
}

func getAllTables(ctx context.Context, extraNativeDriver *ydb.Driver, pathPrefix string) (map[string]struct{}, error) {
	directory, err := extraNativeDriver.Scheme().ListDirectory(ctx, pathPrefix)
	if err != nil {
		return nil, err
	}

	tables := make(map[string]struct{})

	for _, elem := range directory.Children {
		if elem.IsTable() {
			tables[elem.Name] = struct{}{}
		}
	}

	return tables, nil
}
