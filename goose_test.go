package goose

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestDefaultBinary(t *testing.T) {
	t.Parallel()

	commands := []string{
		"go build -o ./bin/goose ./cmd/goose",
		"./bin/goose -dir=examples/sql-migrations sqlite3 sql.db up",
		"./bin/goose -dir=examples/sql-migrations sqlite3 sql.db version",
		"./bin/goose -dir=examples/sql-migrations sqlite3 sql.db down",
		"./bin/goose -dir=examples/sql-migrations sqlite3 sql.db status",
		"./bin/goose --version",
	}
	defer os.Remove("./bin/goose") // clean up

	for _, cmd := range commands {
		args := strings.Split(cmd, " ")
		command := args[0]
		var params []string
		if len(args) > 1 {
			params = args[1:]
		}

		cmd := exec.Command(command, params...)
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s:\n%v\n\n%s", err, cmd, out)
		}
	}
}

func TestLiteBinary(t *testing.T) {
	t.Parallel()

	dir, err := ioutil.TempDir("", "tmptest")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(dir)             // clean up
	defer os.Remove("./bin/lite-goose") // clean up

	// this has to be done outside of the loop
	// since go only supports space separated tags list.
	cmd := exec.Command("go", "build", "-tags='no_postgres no_mysql no_sqlite3'", "-o", "./bin/lite-goose", "./cmd/goose")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s:\n%v\n\n%s", err, cmd, out)
	}

	commands := []string{
		fmt.Sprintf("./bin/lite-goose -dir=%s create user_indices sql", dir),
		fmt.Sprintf("./bin/lite-goose -dir=%s fix", dir),
	}

	for _, cmd := range commands {
		args := strings.Split(cmd, " ")
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s:\n%v\n\n%s", err, cmd, out)
		}
	}
}

func TestCustomBinary(t *testing.T) {
	t.Parallel()

	commands := []string{
		"go build -o ./bin/custom-goose ./examples/go-migrations",
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

//go:embed examples/sql-migrations/*.sql
var migrations embed.FS

func TestEmbeddedMigrations(t *testing.T) {
	// not using t.Parallel here to avoid races

	db, err := sql.Open("sqlite3", "sql_embed.db")
	if err != nil {
		t.Fatalf("Database open failed: %s", err)
	}

	db.SetMaxOpenConns(1)

	// decouple from existing structure
	fsys, err := fs.Sub(migrations, "examples/sql-migrations")
	if err != nil {
		t.Fatalf("SubFS make failed: %s", err)
	}

	SetBaseFS(fsys)
	SetDialect("sqlite3")
	t.Cleanup(func() { SetBaseFS(nil) })

	t.Run("Migration cycle", func(t *testing.T) {
		if err := Up(db, ""); err != nil {
			t.Errorf("Failed to run 'up' migrations: %s", err)
		}

		ver, err := GetDBVersion(db)
		if err != nil {
			t.Fatalf("Failed to get migrations version: %s", err)
		}

		if ver != 3 {
			t.Errorf("Expected version 3 after 'up', got %d", ver)
		}

		if err := Reset(db, ""); err != nil {
			t.Errorf("Failed to run 'down' migrations: %s", err)
		}

		ver, err = GetDBVersion(db)
		if err != nil {
			t.Fatalf("Failed to get migrations version: %s", err)
		}

		if ver != 0 {
			t.Errorf("Expected version 0 after 'reset', got %d", ver)
		}
	})

	t.Run("Create uses os fs", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test_create_osfs")
		if err != nil {
			t.Fatalf("Create temp dir failed: %s", err)
		}

		t.Cleanup(func() { os.RemoveAll(tmpDir) })

		if err := Create(db, tmpDir, "test", "sql"); err != nil {
			t.Errorf("Failed to create migration: %s", err)
		}

		paths, _ := filepath.Glob(filepath.Join(tmpDir, "*test.sql"))
		if len(paths) == 0 {
			t.Errorf("Failed to find created migration")
		}

		if err := Fix(tmpDir); err != nil {
			t.Errorf("Failed to 'fix' migrations: %s", err)
		}

		_, err = os.Stat(filepath.Join(tmpDir, "00001_test.sql"))
		if err != nil {
			t.Errorf("Failed to locate fixed migration: %s", err)
		}
	})
}
