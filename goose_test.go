package goose

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pressly/goose/v3/internal/check"
	_ "modernc.org/sqlite"
)

const (
	binName = "goose-test"
)

func TestMain(m *testing.M) {
	if runtime.GOOS == "windows" {
		log.Fatal("this test is not supported on Windows")
	}
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	args := []string{
		"build",
		"-ldflags=-s -w",
		// disable all drivers except sqlite3
		"-tags=no_clickhouse no_mssql no_mysql no_vertica no_postgres",
		"-o", binName,
		"./cmd/goose",
	}
	build := exec.Command("go", args...)
	out, err := build.CombinedOutput()
	if err != nil {
		log.Fatalf("failed to build %s binary: %v: %s", binName, err, string(out))
	}
	result := m.Run()
	defer func() { os.Exit(result) }()
	if err := os.Remove(filepath.Join(dir, binName)); err != nil {
		log.Printf("failed to remove binary: %v", err)
	}
}

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
	t.Cleanup(func() {
		if err := os.Remove("./bin/goose"); err != nil {
			t.Logf("failed to remove %s resources: %v", t.Name(), err)
		}
		if err := os.Remove("./sql.db"); err != nil {
			t.Logf("failed to remove %s resources: %v", t.Name(), err)
		}
	})

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

func TestIssue532(t *testing.T) {
	t.Parallel()

	migrationsDir := filepath.Join("examples", "sql-migrations")
	count := countSQLFiles(t, migrationsDir)
	check.Number(t, count, 3)

	tempDir := t.TempDir()
	dirFlag := "--dir=" + migrationsDir

	tt := []struct {
		command string
		output  string
	}{
		{"up", ""},
		{"up", "goose: no migrations to run. current version: 3"},
		{"version", "goose: version 3"},
	}
	for _, tc := range tt {
		params := []string{dirFlag, "sqlite3", filepath.Join(tempDir, "sql.db"), tc.command}
		got, err := runGoose(params...)
		check.NoError(t, err)
		if tc.output == "" {
			continue
		}
		if !strings.Contains(strings.TrimSpace(got), tc.output) {
			t.Logf("output mismatch for command: %q", tc.command)
			t.Logf("got\n%s", strings.TrimSpace(got))
			t.Log("====")
			t.Logf("want\n%s", tc.output)
			t.FailNow()
		}
	}
}

func TestIssue293(t *testing.T) {
	t.Parallel()
	// https://github.com/pressly/goose/issues/293
	commands := []string{
		"go build -o ./bin/goose293 ./cmd/goose",
		"./bin/goose293 -dir=examples/sql-migrations sqlite3 issue_293.db up",
		"./bin/goose293 -dir=examples/sql-migrations sqlite3 issue_293.db version",
		"./bin/goose293 -dir=examples/sql-migrations sqlite3 issue_293.db down",
		"./bin/goose293 -dir=examples/sql-migrations sqlite3 issue_293.db down",
		"./bin/goose293 -dir=examples/sql-migrations sqlite3 issue_293.db version",
		"./bin/goose293 -dir=examples/sql-migrations sqlite3 issue_293.db up",
		"./bin/goose293 -dir=examples/sql-migrations sqlite3 issue_293.db status",
	}
	t.Cleanup(func() {
		if err := os.Remove("./bin/goose293"); err != nil {
			t.Logf("failed to remove %s resources: %v", t.Name(), err)
		}
		if err := os.Remove("./issue_293.db"); err != nil {
			t.Logf("failed to remove %s resources: %v", t.Name(), err)
		}
	})
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

func TestIssue336(t *testing.T) {
	t.Parallel()
	// error when no migrations are found
	// https://github.com/pressly/goose/issues/336

	tempDir := t.TempDir()
	params := []string{"--dir=" + tempDir, "sqlite3", filepath.Join(tempDir, "sql.db"), "up"}

	_, err := runGoose(params...)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "no migration files found")
}

func TestLiteBinary(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	t.Cleanup(func() {
		if err := os.Remove("./bin/lite-goose"); err != nil {
			t.Logf("failed to remove %s resources: %v", t.Name(), err)
		}
	})

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
	t.Cleanup(func() {
		if err := os.Remove("./go.db"); err != nil {
			t.Logf("failed to remove %s resources: %v", t.Name(), err)
		}
	})

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
	db, err := sql.Open("sqlite", "sql_embed.db")
	if err != nil {
		t.Fatalf("Database open failed: %s", err)
	}
	t.Cleanup(func() {
		if err := os.Remove("./sql_embed.db"); err != nil {
			t.Logf("failed to remove %s resources: %v", t.Name(), err)
		}
	})

	db.SetMaxOpenConns(1)

	// decouple from existing structure
	fsys, err := fs.Sub(migrations, "examples/sql-migrations")
	if err != nil {
		t.Fatalf("SubFS make failed: %s", err)
	}

	SetBaseFS(fsys)
	check.NoError(t, SetDialect("sqlite3"))
	t.Cleanup(func() { SetBaseFS(nil) })

	t.Run("Migration cycle", func(t *testing.T) {
		if err := Up(db, "."); err != nil {
			t.Errorf("Failed to run 'up' migrations: %s", err)
		}

		ver, err := GetDBVersion(db)
		if err != nil {
			t.Fatalf("Failed to get migrations version: %s", err)
		}

		if ver != 3 {
			t.Errorf("Expected version 3 after 'up', got %d", ver)
		}

		if err := Reset(db, "."); err != nil {
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
		tmpDir := t.TempDir()

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

func runGoose(params ...string) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	cmdPath := filepath.Join(dir, binName)
	cmd := exec.Command(cmdPath, params...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v\n%v", err, string(out))
	}
	return string(out), nil
}

func countSQLFiles(t *testing.T, dir string) int {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	check.NoError(t, err)
	return len(files)
}
