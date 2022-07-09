package e2e

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
)

func TestMigrateUpWithReset(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := newDockerDB(t)
	check.NoError(t, err)

	p, err := goose.NewProvider(toDialect(t, *dialect), db, migrationsDir, nil)
	check.NoError(t, err)
	check.NumberNotZero(t, len(p.ListMigrations()))
	migrations := p.ListMigrations()

	// Migrate all
	err = p.Up(ctx)
	check.NoError(t, err)
	currentVersion, err := p.CurrentVersion(ctx)
	check.NoError(t, err)
	check.Number(t, currentVersion, migrations[len(migrations)-1].Version)
	// Validate the db migration version actually matches what goose claims it is
	// based on the last migration applied.
	gotVersion, err := getCurrentGooseVersion(db, defaultTableName)
	check.NoError(t, err)

	check.Number(t, gotVersion, currentVersion)

	// Migrate everything down using Reset.
	err = p.Reset(ctx)
	check.NoError(t, err)
	currentVersion, err = p.CurrentVersion(ctx)
	check.NoError(t, err)
	check.Number(t, currentVersion, 0)
}

func TestMigrateUpWithRedo(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := newDockerDB(t)
	check.NoError(t, err)

	p, err := goose.NewProvider(toDialect(t, *dialect), db, migrationsDir, nil)
	check.NoError(t, err)

	currentVersion, err := p.CurrentVersion(ctx)
	check.NoError(t, err)
	check.Number(t, currentVersion, 0)

	var count int
	for {
		count++
		err := p.UpByOne(ctx)
		if errors.Is(err, goose.ErrNoNextVersion) {
			break
		}
		if err != nil {
			t.Fatalf("expecting error:%v", goose.ErrNoNextVersion)
		}

		currentVersion, err := p.CurrentVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, count)

		// Redo the previous Up migration and re-apply it.
		err = p.Redo(ctx)
		check.NoError(t, err)

		currentVersion, err = p.CurrentVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, count)
	}
	migrations := p.ListMigrations()
	// Once everything is tested the version should match the highest testdata version
	maxVersion := migrations[len(migrations)-1].Version
	currentVersion, err = p.CurrentVersion(ctx)
	check.NoError(t, err)
	check.Number(t, currentVersion, maxVersion)
}

func TestMigrateUpTo(t *testing.T) {
	t.Parallel()

	const (
		upToVersion int64 = 2
	)
	ctx := context.Background()
	db, err := newDockerDB(t)
	check.NoError(t, err)

	p, err := goose.NewProvider(toDialect(t, *dialect), db, migrationsDir, nil)
	check.NoError(t, err)
	// Migrate up to the second migration
	err = p.UpTo(ctx, upToVersion)
	check.NoError(t, err)
	// Fetch the goose version from DB
	currentVersion, err := p.CurrentVersion(ctx)
	check.NoError(t, err)
	check.Number(t, currentVersion, upToVersion)
	// Validate the version actually matches what goose claims it is
	gotVersion, err := getCurrentGooseVersion(db, defaultTableName)
	check.NoError(t, err)
	check.Number(t, gotVersion, upToVersion)
}

func TestMigrateUpByOne(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := newDockerDB(t)
	check.NoError(t, err)

	p, err := goose.NewProvider(toDialect(t, *dialect), db, migrationsDir, nil)
	check.NoError(t, err)

	migrations := p.ListMigrations()
	// Apply all migrations one-by-one.
	var counter int
	for {
		err := p.UpByOne(ctx)
		if errors.Is(err, goose.ErrNoNextVersion) {
			break
		}
		if err != nil && !errors.Is(err, goose.ErrNoNextVersion) {
			t.Fatalf("unexpected error:%v", err)
		}
		check.NoError(t, err)
		if counter > len(migrations) {
			t.Fatalf("counter exceeds number of migrations")
		}
		counter++
	}
	currentVersion, err := p.CurrentVersion(ctx)
	check.NoError(t, err)
	check.Number(t, currentVersion, migrations[len(migrations)-1].Version)
	// Validate the db migration version actually matches what goose claims it is
	gotVersion, err := getCurrentGooseVersion(db, defaultTableName)
	check.NoError(t, err)
	check.Number(t, gotVersion, currentVersion)
}

func TestMigrateFull(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db, err := newDockerDB(t)
	check.NoError(t, err)

	p, err := goose.NewProvider(toDialect(t, *dialect), db, migrationsDir, nil)
	check.NoError(t, err)

	migrations := p.ListMigrations()
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
		err = p.Up(ctx)
		check.NoError(t, err)
		currentVersion, err := p.CurrentVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, migrations[len(migrations)-1].Version)
		// Validate the db migration version actually matches what goose claims it is
		gotVersion, err := getCurrentGooseVersion(db, defaultTableName)
		check.NoError(t, err)
		check.Number(t, gotVersion, currentVersion)
		tables, err := getTableNames(db)
		check.NoError(t, err)
		if !reflect.DeepEqual(tables, knownTables) {
			t.Logf("got tables: %v", tables)
			t.Logf("known tables: %v", knownTables)
			t.Fatal("failed to match tables")
		}
	}
	{
		// Apply 1 down migration
		err := p.Down(ctx)
		check.NoError(t, err)
		gotVersion, err := getCurrentGooseVersion(db, defaultTableName)
		check.NoError(t, err)
		check.Number(t, gotVersion, migrations[len(migrations)-1].Version-1)
	}
	{
		// Migrate all remaining migrations down. Should only be left with a single table:
		// the default goose table
		err := p.DownTo(ctx, 0)
		check.NoError(t, err)
		gotVersion, err := getCurrentGooseVersion(db, defaultTableName)
		check.NoError(t, err)
		check.Number(t, gotVersion, 0)
		tables, err := getTableNames(db)
		check.NoError(t, err)
		knownTables := []string{defaultTableName}
		if !reflect.DeepEqual(tables, knownTables) {
			t.Logf("got tables: %v", tables)
			t.Logf("known tables: %v", knownTables)
			t.Fatal("failed to match tables")
		}
	}
}

func getCurrentGooseVersion(db *sql.DB, gooseTable string) (int64, error) {
	var gotVersion int64
	if err := db.QueryRow(
		fmt.Sprintf("select max(version_id) from %s", gooseTable),
	).Scan(&gotVersion); err != nil {
		return 0, err
	}
	return gotVersion, nil
}

func getGooseVersionCount(db *sql.DB, gooseTable string) (int64, error) {
	var gotVersion int64
	if err := db.QueryRow(
		fmt.Sprintf("SELECT count(*) FROM %s WHERE version_id > 0", gooseTable),
	).Scan(&gotVersion); err != nil {
		return 0, err
	}
	return gotVersion, nil
}

func getTableNames(db *sql.DB) ([]string, error) {
	var query string
	switch *dialect {
	case dialectPostgres:
		query = `SELECT table_name FROM information_schema.tables WHERE table_schema='public' ORDER BY table_name`
	case dialectMySQL:
		query = `SELECT table_name FROM INFORMATION_SCHEMA.tables WHERE TABLE_TYPE='BASE TABLE' ORDER BY table_name`
	}
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tableNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tableNames = append(tableNames, name)
	}
	return tableNames, nil
}
