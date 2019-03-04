package goose

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestDefaultBinary(t *testing.T) {
	commands := []string{
		"go build -i -o ./bin/goose ./cmd/goose",
		"./bin/goose -dir=examples/sql-migrations sqlite3 sql.db up",
		"./bin/goose -dir=examples/sql-migrations sqlite3 sql.db version",
		"./bin/goose -dir=examples/sql-migrations sqlite3 sql.db down",
		"./bin/goose -dir=examples/sql-migrations sqlite3 sql.db status",
		"./bin/goose",
	}

	for _, cmd := range commands {
		args := strings.Split(cmd, " ")
		command := args[0]
		var params []string
		if len(args) > 1 {
			params = args[1:]
		}

		out, err := exec.Command(command, params...).CombinedOutput()
		if err != nil {
			t.Fatalf("%s:\n%v\n\n%s", err, cmd, out)
		}
	}
}

func TestLiteBinary(t *testing.T) {
	dir, err := ioutil.TempDir("", "tmptest")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(dir)        // clean up
	defer os.Remove("./bin/goose") // clean up

	commands := []string{
		fmt.Sprintf("./bin/goose -dir=%s create user_indices sql", dir),
		fmt.Sprintf("./bin/goose -dir=%s fix", dir),
	}

	// this has to be done outside of the loop
	// since go only supports space separated tags list.
	cmd := "go build -tags='no_mysql no_sqlite no_psql' -i -o ./bin/goose ./cmd/goose"
	out, err := exec.Command("go", "build", "-tags='no_mysql no_sqlite no_psql'", "-i", "-o", "./bin/goose", "./cmd/goose").CombinedOutput()
	if err != nil {
		t.Fatalf("%s:\n%v\n\n%s", err, cmd, out)
	}

	for _, cmd := range commands {
		args := strings.Split(cmd, " ")
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			t.Fatalf("%s:\n%v\n\n%s", err, cmd, out)
		}
	}
}

func TestCustomBinary(t *testing.T) {
	commands := []string{
		"go build -i -o ./bin/custom-goose ./examples/go-migrations",
		"./bin/custom-goose -dir=examples/go-migrations sqlite3 go.db up",
		"./bin/custom-goose -dir=examples/go-migrations sqlite3 go.db version",
		"./bin/custom-goose -dir=examples/go-migrations sqlite3 go.db down",
		"./bin/custom-goose -dir=examples/go-migrations sqlite3 go.db status",
	}

	for _, cmd := range commands {
		args := strings.Split(cmd, " ")
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			t.Fatalf("%s:\n%v\n\n%s", err, cmd, out)
		}
	}
}
