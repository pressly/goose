package goose

import (
	"os/exec"
	"strings"
	"testing"
)

func TestDefaultBinary(t *testing.T) {
	commands := []string{
		"go build -i -o goose ./cmd/goose",
		"./goose -dir=examples/sql-migrations sqlite3 sql.db up",
		"./goose -dir=examples/sql-migrations sqlite3 sql.db version",
		"./goose -dir=examples/sql-migrations sqlite3 sql.db down",
		"./goose -dir=examples/sql-migrations sqlite3 sql.db status",
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
		"go build -i -o custom-goose ./examples/go-migrations",
		"./custom-goose -dir=examples/go-migrations sqlite3 go.db up",
		"./custom-goose -dir=examples/go-migrations sqlite3 go.db version",
		"./custom-goose -dir=examples/go-migrations sqlite3 go.db down",
		"./custom-goose -dir=examples/go-migrations sqlite3 go.db status",
	}

	for _, cmd := range commands {
		args := strings.Split(cmd, " ")
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			t.Fatalf("%s:\n%v\n\n%s", err, cmd, out)
		}
	}
}
