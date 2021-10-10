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
	// Validate the version actually matches what goose claims it is .
	var gotVersion int
	err = db.QueryRow(
		fmt.Sprintf("select max(version_id) from %s", goose.TableName()),
	).Scan(&gotVersion)
	is.NoErr(err)
	is.Equal(gotVersion, numberOfMigrations) // incorrect database version.
}

func TestMigrateUpTo(t *testing.T) {
	const (
		upToVersion = 2
	)
	db, err := newDockerDatabase(t, dialectPostgres, 0)
	if err != nil {
		t.Fatal(err)
	}
	goose.SetDialect(dialectPostgres)
	migrationsDir := filepath.Join("testdata", "migrations")
	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	if err != nil {
		t.Fatal(err)
	}
	numberOfMigrations := len(migrations)
	if numberOfMigrations == 0 {
		t.Fatal("must supply valid migrations")
	}
	// Migrate up to the second migration.
	if err := goose.UpTo(db, migrationsDir, upToVersion); err != nil {
		t.Fatal(err)
	}
	currentVersion, err := goose.GetDBVersion(db)
	if err != nil {
		t.Fatal(err)
	}
	if currentVersion != upToVersion {
		t.Fatalf("failed to match current vesion:%d with migration file max version:%d",
			currentVersion,
			upToVersion,
		)
	}
	// Validate the version actually matches what goose claims it is .
	row := db.QueryRow(fmt.Sprintf("select max(version_id) from %s", goose.TableName()))
	var gotVersion int
	if err := row.Scan(&gotVersion); err != nil {
		t.Fatal(err)
	}
	if gotVersion != upToVersion {
		t.Fatalf("failed to match db version with migration number from dir: got %d want %d",
			gotVersion,
			upToVersion,
		)
	}
}
