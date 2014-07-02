package goose

import (
	"os"
	"reflect"
	"testing"
)

func TestBasics(t *testing.T) {

	dbconf, err := NewDBConf("../../db-sample", "test", "")
	if err != nil {
		t.Fatal(err)
	}

	got := []string{dbconf.MigrationsDir, dbconf.Env, dbconf.Driver.Name, dbconf.Driver.OpenStr}
	want := []string{"../../db-sample/migrations", "test", "postgres", "user=liam dbname=tester sslmode=disable"}

	for i, s := range got {
		if s != want[i] {
			t.Errorf("Unexpected DBConf value. got %v, want %v", s, want[i])
		}
	}
}

func TestImportOverride(t *testing.T) {

	dbconf, err := NewDBConf("../../db-sample", "customimport", "")
	if err != nil {
		t.Fatal(err)
	}

	got := dbconf.Driver.Import
	want := "github.com/custom/driver"
	if got != want {
		t.Errorf("bad custom import. got %v want %v", got, want)
	}
}

func TestDriverSetFromEnvironmentVariable(t *testing.T) {

	databaseUrlEnvVariableKey := "DB_DRIVER"
	databaseUrlEnvVariableVal := "sqlite3"
	databaseOpenStringKey := "DATABASE_URL"
	databaseOpenStringVal := "db.db"

	os.Setenv(databaseUrlEnvVariableKey, databaseUrlEnvVariableVal)
	os.Setenv(databaseOpenStringKey, databaseOpenStringVal)

	dbconf, err := NewDBConf("../../db-sample", "environment_variable_config", "")
	if err != nil {
		t.Fatal(err)
	}

	got := reflect.TypeOf(dbconf.Driver.Dialect)
	want := reflect.TypeOf(&Sqlite3Dialect{})

	if got != want {
		t.Errorf("Not able to read the driver type from environment variable."+
			"got %v want %v", got, want)
	}

	gotOpenString := dbconf.Driver.OpenStr
	wantOpenString := databaseOpenStringVal

	if gotOpenString != wantOpenString {
		t.Errorf("Not able to read the open string from the environment."+
			"got %v want %v", gotOpenString, wantOpenString)
	}
}
