package main

import (
	// "fmt"
	"testing"
)

func TestBasics(t *testing.T) {

	dbconf, err := makeDBConfDetails("db-sample", "test")
	if err != nil {
		t.Error("couldn't create DBConf")
	}

	got := []string{dbconf.MigrationsDir, dbconf.Env, dbconf.Driver.Name, dbconf.Driver.OpenStr}
	want := []string{"db-sample/migrations", "test", "postgres", "user=liam dbname=tester sslmode=disable"}

	for i, s := range got {
		if s != want[i] {
			t.Errorf("Unexpected DBConf value. got %v, want %v", s, want[i])
		}
	}
}

func TestImportOverride(t *testing.T) {

	dbconf, err := makeDBConfDetails("db-sample", "customimport")
	if err != nil {
		t.Error("couldn't create DBConf")
	}

	got := dbconf.Driver.Import
	want := "github.com/custom/driver"
	if got != want {
		t.Errorf("bad custom import. want %v got %v", got, want)
	}
}
