package e2e

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/matryer/is"
	"github.com/pressly/goose/v3"
)

func TestMigrateUpWithReset(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	db, err := newDockerDB(t)
	is.NoErr(err)
	goose.SetDialect(*dialect)
	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	is.NoErr(err)
	is.True(len(migrations) != 0)
	// Migrate all
	err = goose.Up(db, migrationsDir)
	is.NoErr(err)
	currentVersion, err := goose.GetDBVersion(db)
	is.NoErr(err)
	is.Equal(currentVersion, migrations[len(migrations)-1].Version)
	// Validate the db migration version actually matches what goose claims it is
	gotVersion, err := getCurrentGooseVersion(db, goose.TableName())
	is.NoErr(err)
	is.Equal(gotVersion, currentVersion) // incorrect database version

	// Migrate everything down using Reset.
	err = goose.Reset(db, migrationsDir)
	is.NoErr(err)
	currentVersion, err = goose.GetDBVersion(db)
	is.NoErr(err)
	is.Equal(currentVersion, int64(0))
}

func TestMigrateUpWithRedo(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	db, err := newDockerDB(t)
	is.NoErr(err)
	goose.SetDialect(*dialect)
	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	is.NoErr(err)
	is.True(len(migrations) != 0)
	startingVersion, err := goose.EnsureDBVersion(db)
	is.NoErr(err)
	is.Equal(startingVersion, int64(0))
	// Migrate all
	for _, migration := range migrations {
		err = migration.Up(db)
		is.NoErr(err)
		currentVersion, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.True(currentVersion == migration.Version)

		// Redo the previous Up migration and re-apply it.
		err = goose.Redo(db, migrationsDir)
		is.NoErr(err)
		currentVersion, err = goose.GetDBVersion(db)
		is.NoErr(err)
		is.True(currentVersion == migration.Version)
	}
	// Once everything is tested the version should match the highest testdata version
	maxVersion := migrations[len(migrations)-1].Version
	currentVersion, err := goose.GetDBVersion(db)
	is.NoErr(err)
	is.Equal(currentVersion, maxVersion)
}

func TestMigrateUpTo(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	const (
		upToVersion int64 = 2
	)
	db, err := newDockerDB(t)
	is.NoErr(err)
	goose.SetDialect(*dialect)
	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	is.NoErr(err)
	is.True(len(migrations) != 0)
	// Migrate up to the second migration
	err = goose.UpTo(db, migrationsDir, upToVersion)
	is.NoErr(err)
	// Fetch the goose version from DB
	currentVersion, err := goose.GetDBVersion(db)
	is.NoErr(err)
	is.Equal(currentVersion, upToVersion)
	// Validate the version actually matches what goose claims it is
	gotVersion, err := getCurrentGooseVersion(db, goose.TableName())
	is.NoErr(err)
	is.Equal(gotVersion, upToVersion) // incorrect database version
}

func TestMigrateUpByOne(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	db, err := newDockerDB(t)
	is.NoErr(err)
	goose.SetDialect(*dialect)
	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	is.NoErr(err)
	is.True(len(migrations) != 0)
	// Apply all migrations one-by-one.
	var counter int
	for {
		err := goose.UpByOne(db, migrationsDir)
		counter++
		if counter > len(migrations) {
			is.Equal(err, goose.ErrNoNextVersion)
			break
		}
		is.NoErr(err)
	}
	currentVersion, err := goose.GetDBVersion(db)
	is.NoErr(err)
	is.Equal(currentVersion, migrations[len(migrations)-1].Version)
	// Validate the db migration version actually matches what goose claims it is
	gotVersion, err := getCurrentGooseVersion(db, goose.TableName())
	is.NoErr(err)
	is.Equal(gotVersion, currentVersion) // incorrect database version
}

func TestMigrateFull(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	db, err := newDockerDB(t)
	is.NoErr(err)
	goose.SetDialect(*dialect)
	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	is.NoErr(err)
	is.True(len(migrations) != 0)
	// test retrieving invalid current goose migrations. goose cannot return a migration
	// if the supplied "current" version is non-existent or 0.
	_, err = migrations.Current(20160813)
	is.Equal(err, goose.ErrNoCurrentVersion)
	_, err = migrations.Current(0)
	is.Equal(err, goose.ErrNoCurrentVersion)
	// verify the first migration1. This should not change if more migrations are added
	// in the future.
	migration1, err := migrations.Current(1)
	is.NoErr(err)
	is.Equal(migration1.Version, int64(1))
	is.Equal(migration1.Source, filepath.Join(migrationsDir, "00001_a.sql"))
	is.Equal(migration1.Registered, false) // expecting false for .sql migrations
	is.Equal(migration1.Previous, int64(-1))
	is.Equal(migration1.Next, int64(2))

	{
		// Apply all up migrations
		err = goose.Up(db, migrationsDir)
		is.NoErr(err)
		currentVersion, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(currentVersion, migrations[len(migrations)-1].Version)
		// Validate the db migration version actually matches what goose claims it is
		gotVersion, err := getCurrentGooseVersion(db, goose.TableName())
		is.NoErr(err)
		is.Equal(gotVersion, currentVersion) // incorrect database version
		tables, err := getTableNames(db)
		is.NoErr(err)
		is.Equal(tables, knownTables)
	}
	{
		// Apply 1 down migration
		err := goose.Down(db, migrationsDir)
		is.NoErr(err)
		gotVersion, err := getCurrentGooseVersion(db, goose.TableName())
		is.NoErr(err)
		is.Equal(gotVersion, migrations[len(migrations)-1].Version-1) // incorrect database version
	}
	{
		// Migrate all remaining migrations down. Should only be left with a single table:
		// the default goose table
		err := goose.DownTo(db, migrationsDir, 0)
		is.NoErr(err)
		gotVersion, err := getCurrentGooseVersion(db, goose.TableName())
		is.NoErr(err)
		is.Equal(gotVersion, int64(0))
		tables, err := getTableNames(db)
		is.NoErr(err)
		is.Equal(tables, []string{goose.TableName()})
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
