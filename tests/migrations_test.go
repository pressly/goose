package postgres_test

import (
	"fmt"
	"path/filepath"
	"testing"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

func TestMigrateUp(t *testing.T) {
	db, err := newDockerDatabase(t, dialectPostgres, 5432)
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
	// Migrate all
	if err := goose.Up(db, migrationsDir); err != nil {
		t.Fatal(err)
	}
	currentVersion, err := goose.GetDBVersion(db)
	if err != nil {
		t.Fatal(err)
	}
	if want := migrations[len(migrations)-1].Version; currentVersion != want {
		t.Fatalf("failed to match current vesion:%d with migration file max version:%d",
			currentVersion,
			want,
		)
	}
	// Validate the version actually matches what goose claims it is .
	row := db.QueryRow(fmt.Sprintf("select max(version_id) from %s", goose.TableName()))
	var gotVersion int
	if err := row.Scan(&gotVersion); err != nil {
		t.Fatal(err)
	}
	if gotVersion != numberOfMigrations {
		t.Fatalf("failed to match db version with migration number from dir: got %d want %d",
			gotVersion,
			numberOfMigrations,
		)
	}
}

func TestMigrateUpTo(t *testing.T) {
	const (
		upToVersion = 2
	)
	db, err := newDockerDatabase(t, dialectPostgres, 5432)
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
