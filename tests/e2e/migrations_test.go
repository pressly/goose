package e2e

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
)

func TestMigrateUpWithReset(t *testing.T) {
	t.Parallel()

	db, err := newDockerDB(t)
	check.NoError(t, err)
	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	check.NoError(t, err)
	check.NumberNotZero(t, len(migrations))
	// Migrate all
	err = goose.Up(db, migrationsDir)
	check.NoError(t, err)
	currentVersion, err := goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, migrations[len(migrations)-1].Version)
	// Validate the db migration version actually matches what goose claims it is
	gotVersion, err := getCurrentGooseVersion(db, goose.TableName())
	check.NoError(t, err)
	// incorrect database version
	check.Number(t, gotVersion, currentVersion)

	// Migrate everything down using Reset.
	err = goose.Reset(db, migrationsDir)
	check.NoError(t, err)
	currentVersion, err = goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, 0)
}

func TestMigrateUpWithRedo(t *testing.T) {
	t.Parallel()

	db, err := newDockerDB(t)
	check.NoError(t, err)
	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	check.NoError(t, err)

	check.NumberNotZero(t, len(migrations))
	startingVersion, err := goose.EnsureDBVersion(db)
	check.NoError(t, err)
	check.Number(t, startingVersion, 0)
	// Migrate all
	for _, migration := range migrations {
		err = migration.Up(db)
		check.NoError(t, err)
		currentVersion, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, currentVersion, migration.Version)

		// Redo the previous Up migration and re-apply it.
		err = goose.Redo(db, migrationsDir)
		check.NoError(t, err)
		currentVersion, err = goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, currentVersion, migration.Version)
	}
	// Once everything is tested the version should match the highest testdata version
	maxVersion := migrations[len(migrations)-1].Version
	currentVersion, err := goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, maxVersion)
}

func TestMigrateUpTo(t *testing.T) {
	t.Parallel()

	const (
		upToVersion int64 = 2
	)
	db, err := newDockerDB(t)
	check.NoError(t, err)
	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	check.NoError(t, err)
	check.NumberNotZero(t, len(migrations))
	// Migrate up to the second migration
	err = goose.UpTo(db, migrationsDir, upToVersion)
	check.NoError(t, err)
	// Fetch the goose version from DB
	currentVersion, err := goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, upToVersion)
	// Validate the version actually matches what goose claims it is
	gotVersion, err := getCurrentGooseVersion(db, goose.TableName())
	check.NoError(t, err)
	check.Number(t, gotVersion, upToVersion) // incorrect database version
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %q: %v", name, err)
	}
}

func TestMigrateUpTimeout(t *testing.T) {
	t.Parallel()
	if *dialect != dialectPostgres {
		t.Skipf("skipping test for dialect: %q", *dialect)
	}

	dir := t.TempDir()
	writeFile(t, dir, "00001_a.sql", `
-- +goose Up
SELECT 1;
`)
	writeFile(t, dir, "00002_a.sql", `
-- +goose Up
SELECT pg_sleep(10);
`)
	db, err := newDockerDB(t)
	check.NoError(t, err)
	// Simulate timeout midway through a set of migrations. This should leave the
	// database in a state where it has applied some migrations, but not all.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	migrations, err := goose.CollectMigrations(dir, 0, goose.MaxVersion)
	check.NoError(t, err)
	check.NumberNotZero(t, len(migrations))
	// Apply all migrations.
	err = goose.UpContext(ctx, db, dir)
	check.HasError(t, err) // expect it to timeout.
	check.Bool(t, errors.Is(err, context.DeadlineExceeded), true)

	currentVersion, err := goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, 1)
	// Validate the db migration version actually matches what goose claims it is
	gotVersion, err := getCurrentGooseVersion(db, goose.TableName())
	check.NoError(t, err)
	check.Number(t, gotVersion, 1)
}

func TestMigrateDownTimeout(t *testing.T) {
	t.Parallel()
	if *dialect != dialectPostgres {
		t.Skipf("skipping test for dialect: %q", *dialect)
	}
	dir := t.TempDir()
	writeFile(t, dir, "00001_a.sql", `
-- +goose Up
SELECT 1;
-- +goose Down
SELECT pg_sleep(10);
`)
	writeFile(t, dir, "00002_a.sql", `
-- +goose Up
SELECT 1;
`)
	db, err := newDockerDB(t)
	check.NoError(t, err)
	// Simulate timeout midway through a set of migrations. This should leave the
	// database in a state where it has applied some migrations, but not all.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	migrations, err := goose.CollectMigrations(dir, 0, goose.MaxVersion)
	check.NoError(t, err)
	check.NumberNotZero(t, len(migrations))
	// Apply all up migrations.
	err = goose.UpContext(ctx, db, dir)
	check.NoError(t, err)
	// Applly all down migrations.
	err = goose.DownToContext(ctx, db, dir, 0)
	check.HasError(t, err) // expect it to timeout.
	check.Bool(t, errors.Is(err, context.DeadlineExceeded), true)

	currentVersion, err := goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, 1)
	// Validate the db migration version actually matches what goose claims it is
	gotVersion, err := getCurrentGooseVersion(db, goose.TableName())
	check.NoError(t, err)
	check.Number(t, gotVersion, 1)
}

func TestMigrateUpByOne(t *testing.T) {
	t.Parallel()

	db, err := newDockerDB(t)
	check.NoError(t, err)
	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	check.NoError(t, err)
	check.NumberNotZero(t, len(migrations))
	// Apply all migrations one-by-one.
	var counter int
	for {
		err := goose.UpByOne(db, migrationsDir)
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
	gotVersion, err := getCurrentGooseVersion(db, goose.TableName())
	check.NoError(t, err)
	check.Number(t, gotVersion, currentVersion) // incorrect database version
}

func TestMigrateFull(t *testing.T) {
	t.Parallel()

	db, err := newDockerDB(t)
	check.NoError(t, err)
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
		err = goose.Up(db, migrationsDir)
		check.NoError(t, err)
		currentVersion, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, currentVersion, migrations[len(migrations)-1].Version)
		// Validate the db migration version actually matches what goose claims it is
		gotVersion, err := getCurrentGooseVersion(db, goose.TableName())
		check.NoError(t, err)
		check.Number(t, gotVersion, currentVersion) // incorrect database version
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
		err := goose.Down(db, migrationsDir)
		check.NoError(t, err)
		gotVersion, err := getCurrentGooseVersion(db, goose.TableName())
		check.NoError(t, err)
		check.Number(t, gotVersion, migrations[len(migrations)-1].Version-1) // incorrect database version
	}
	{
		// Migrate all remaining migrations down. Should only be left with a single table:
		// the default goose table
		err := goose.DownTo(db, migrationsDir, 0)
		check.NoError(t, err)
		gotVersion, err := getCurrentGooseVersion(db, goose.TableName())
		check.NoError(t, err)
		check.Number(t, gotVersion, 0)
		tables, err := getTableNames(db)
		check.NoError(t, err)
		knownTables := []string{goose.TableName()}
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

func getTableNamesThroughQuery(db *sql.DB, query string) ([]string, error) {
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

func getTableNames(db *sql.DB) (tableNames []string, _ error) {
	switch *dialect {
	case dialectPostgres:
		return getTableNamesThroughQuery(db,
			`SELECT table_name FROM information_schema.tables WHERE table_schema='public' ORDER BY table_name`,
		)
	case dialectMySQL:
		return getTableNamesThroughQuery(db,
			`SELECT table_name FROM INFORMATION_SCHEMA.tables WHERE TABLE_TYPE='BASE TABLE' ORDER BY table_name`,
		)
	case dialectYdb:
		conn, err := db.Conn(context.Background())
		if err != nil {
			return nil, err
		}
		if err = conn.Raw(func(rawConn any) error {
			if tables, has := rawConn.(interface {
				GetTables(ctx context.Context, folder string, recursive bool, excludeSysDirs bool) (tables []string, err error)
			}); has {
				tableNames, err = tables.GetTables(context.Background(), ".", true, true)
				if err != nil {
					return err
				}
				return nil
			}
			return fmt.Errorf("%T not implemented GetTables interface", rawConn)
		}); err != nil {
			return nil, err
		}
		return tableNames, nil
	case dialectTurso:
		return getTableNamesThroughQuery(db, `SELECT NAME FROM sqlite_master where type='table' and name!='sqlite_sequence' ORDER BY NAME;`)
	case dialectDuckDB:
		return getTableNamesThroughQuery(db,
			`SELECT table_name FROM information_schema.tables ORDER BY table_name`,
		)
	default:
		return nil, fmt.Errorf("getTableNames not supported with dialect %q", *dialect)
	}
}
