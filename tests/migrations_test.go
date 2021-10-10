package postgres_test

import (
	"fmt"
	"path/filepath"
	"testing"

	_ "github.com/lib/pq"
	"github.com/matryer/is"
	"github.com/pressly/goose/v3"
)

func TestMigrateUp(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	db, err := newDockerDatabase(t, dialectPostgres, 0)
	is.NoErr(err)
	goose.SetDialect(dialectPostgres)
	migrationsDir := filepath.Join("testdata", "migrations")
	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	is.NoErr(err)
	numberOfMigrations := len(migrations)
	is.True(numberOfMigrations != 0)
	// Migrate all
	err = goose.Up(db, migrationsDir)
	is.NoErr(err)
	currentVersion, err := goose.GetDBVersion(db)
	is.NoErr(err)
	is.Equal(currentVersion, migrations[len(migrations)-1].Version)
	// Validate the version actually matches what goose claims it is
	var gotVersion int
	err = db.QueryRow(
		fmt.Sprintf("select max(version_id) from %s", goose.TableName()),
	).Scan(&gotVersion)
	is.NoErr(err)
	is.Equal(gotVersion, numberOfMigrations) // incorrect database version
}

func TestMigrateUpTo(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	const (
		upToVersion int64 = 2
	)
	db, err := newDockerDatabase(t, dialectPostgres, 0)
	is.NoErr(err)
	goose.SetDialect(dialectPostgres)
	migrationsDir := filepath.Join("testdata", "migrations")
	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	is.NoErr(err)
	numberOfMigrations := len(migrations)
	is.True(numberOfMigrations != 0)
	// Migrate up to the second migration
	err = goose.UpTo(db, migrationsDir, upToVersion)
	is.NoErr(err)
	// Fetch the goose version from DB
	currentVersion, err := goose.GetDBVersion(db)
	is.NoErr(err)
	is.Equal(currentVersion, upToVersion)
	// Validate the version actually matches what goose claims it is .
	var gotVersion int64
	err = db.QueryRow(
		fmt.Sprintf("select max(version_id) from %s", goose.TableName()),
	).Scan(&gotVersion)
	is.NoErr(err)
	is.Equal(gotVersion, upToVersion) // incorrect database version
}

func TestMigrateUpByOne(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	db, err := newDockerDatabase(t, dialectPostgres, 0)
	is.NoErr(err)
	goose.SetDialect(dialectPostgres)
	migrationsDir := filepath.Join("testdata", "migrations")
	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	is.NoErr(err)
	numberOfMigrations := len(migrations)
	is.True(numberOfMigrations != 0)
	// Migrate up to the second migration

	var counter int
	for {
		err := goose.UpByOne(db, migrationsDir)
		counter++
		if counter > numberOfMigrations {
			is.Equal(err, goose.ErrNoNextVersion)
			break
		}
		is.NoErr(err)
	}
	// Fetch the goose version from DB
	currentVersion, err := goose.GetDBVersion(db)
	is.NoErr(err)
	is.Equal(currentVersion, migrations[len(migrations)-1].Version)
	// Validate the version actually matches what goose claims it is .
	var gotVersion int
	err = db.QueryRow(
		fmt.Sprintf("select max(version_id) from %s", goose.TableName()),
	).Scan(&gotVersion)
	is.NoErr(err)
	is.Equal(gotVersion, numberOfMigrations) // incorrect database version
}
