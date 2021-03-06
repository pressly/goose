package iofs_test

import (
	"database/sql"
	"embed"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/pressly/goose"
	"github.com/pressly/goose/iofs"
)

//go:embed testdata
var testdataFS embed.FS

const (
	migrationsPath = "testdata/migrations"
	migrationsCount = 3
	maxMigrationVersion = 3
)

func TestCollect(t *testing.T) {
	migrations, err := iofs.CollectMigrations(testdataFS, migrationsPath, 0, goose.MaxVersion)
	if err != nil {
		t.Fatalf("Collect migrations failed: %v", err)
	}

	if len(migrations) != migrationsCount {
		t.Errorf("Unexpected number of migrations: %d", len(migrations))
	}
}

func TestMigrationCycle(t *testing.T) {
	db, err := sql.Open("sqlite3", "sql.db")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	defer db.Close()

	db.SetMaxOpenConns(1)

	goose.SetLogger((*tLogger)(t))
	goose.SetDialect("sqlite3")

	if err := iofs.Up(db, testdataFS, migrationsPath); err != nil {
		t.Errorf("Failed to run up migrations: %v", err)
	}

	if err := iofs.Status(db, testdataFS, migrationsPath); err != nil {
		t.Errorf("Failed to print migrations status: %v", err)
	}

	version, err := goose.GetDBVersion(db)
	if err != nil {
		t.Errorf("Failed to get db version: %v", err)
	}

	if version != maxMigrationVersion {
		t.Errorf("Unexpected version after up: %d", version)
	}

	if err := iofs.Down(db, testdataFS, migrationsPath); err != nil {
		t.Errorf("Failed to down one migration: %v", err)
	}

	version, err = goose.GetDBVersion(db)
	if err != nil {
		t.Errorf("Failed to get db version: %v", err)
	}

	if version != maxMigrationVersion - 1 {
		t.Errorf("Unexpected version after down: %d", version)
	}

	if err := iofs.Status(db, testdataFS, migrationsPath); err != nil {
		t.Errorf("Failed to print migrations status: %v", err)
	}
}

type tLogger testing.T

func (t *tLogger) Print(v ...interface{}) { t.Log(v...) }

func (t *tLogger) Println(v ...interface{}) { t.Log(v...) }

func (t *tLogger) Printf(format string, v ...interface{}) { t.Logf(format, v...) }

