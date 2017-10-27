package goose

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestDefaultBinary(t *testing.T) {
	bin := "goose"
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	commands := []string{
		fmt.Sprintf("go build -i -o %s ./cmd/goose", bin),
		fmt.Sprintf("./%s -dir=examples/sql-migrations sqlite3 sql.db up", bin),
		fmt.Sprintf("./%s -dir=examples/sql-migrations sqlite3 sql.db version", bin),
		fmt.Sprintf("./%s -dir=examples/sql-migrations sqlite3 sql.db down", bin),
		fmt.Sprintf("./%s -dir=examples/sql-migrations sqlite3 sql.db status", bin),
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
	bin := "custom-goose"
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	commands := []string{
		fmt.Sprintf("go build -i -o %s ./examples/go-migrations", bin),
		fmt.Sprintf("./%s -dir=examples/go-migrations sqlite3 go.db up", bin),
		fmt.Sprintf("./%s -dir=examples/go-migrations sqlite3 go.db version", bin),
		fmt.Sprintf("./%s -dir=examples/go-migrations sqlite3 go.db down", bin),
		fmt.Sprintf("./%s -dir=examples/go-migrations sqlite3 go.db status", bin),
	}

	for _, cmd := range commands {
		args := strings.Split(cmd, " ")
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			t.Fatalf("%s:\n%v\n\n%s", err, cmd, out)
		}
	}
}
