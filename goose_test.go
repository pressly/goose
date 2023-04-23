package goose_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pressly/goose/v4"
	"github.com/pressly/goose/v4/internal/check"
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
		"-tags=no_postgres no_clickhouse no_mssql no_mysql no_vertica",
		"-o", binName,
		"./cmd/goose",
	}
	build := exec.Command("go", args...)
	if err := build.Run(); err != nil {
		log.Fatalf("failed to build %s binary: %s", binName, err)
	}
	result := m.Run()
	if err := os.Remove(filepath.Join(dir, binName)); err != nil {
		log.Printf("failed to remove binary: %s", err)
	}
	os.Exit(result)
}

func TestBinaryVersion(t *testing.T) {
	t.Parallel()
	out := runGoose(t, "--version")
	check.Contains(t, out, "goose version: (devel)")
}

func TestGooseInit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dirFlag := "--dir=" + filepath.Join(dir, "migrations")
	out := runGoose(t, "init", dirFlag)
	check.Contains(t, out, "00001_initial.sql")
}

func TestGooseCreate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dirFlag := "--dir=" + filepath.Join(dir, "migrations")
	out := runGoose(t, "create", "-s", dirFlag, "sql", "add users table")
	check.Contains(t, out, "00001_add_users_table.sql")
}

func TestDefaultBinary(t *testing.T) {
	t.Parallel()

	migrationsDir := filepath.Join("examples", "sql-migrations")
	count := countSQLFiles(t, migrationsDir)
	check.Number(t, count, 3)

	dirFlag := "--dir=" + migrationsDir
	dbStringFlag := "--dbstring=" + newDBString(t)

	tt := []struct {
		command string
		args    string
		output  string
	}{
		// TODO(mf): check output for empty output test cases
		{"up", "", ""},
		{"version", "", "goose: version  3"},
		{"up", "", "no migrations to run"},
		{"down-to", "0", ""},
		{"version", "", "goose: version  0"},
		{"down-to", "0", "no migrations to run"},
		{"status", "", ""},
	}
	for _, tc := range tt {
		params := []string{tc.command, dirFlag, dbStringFlag}
		params = append(params, strings.Split(tc.args, " ")...)
		got := runGoose(t, params...)
		if tc.output == "" {
			continue
		}
		if strings.TrimSpace(got) != tc.output {
			t.Logf("output mismatch for command: %q", tc.command[0])
			t.Logf("got\n%s", strings.TrimSpace(got))
			t.Log("====")
			t.Logf("want\n%s", tc.output)
			t.FailNow()
		}
	}
}

func TestEmbedBinary(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	/*
		To avoid accidental changes to the embedded migrations, we copy them to a temp dir.

		In a real application, you would use the migrations embedded in your binary.
		For example:

		//go:embed examples/sql-migrations/*.sql
		var migrations embed.FS

		opt := goose.DefaultOptions().
			SetDir("examples/sql-migrations").
			SetFilesystem(migrations)
		provider, err := goose.NewProvider(dialect, db, opt)
	*/

	dir := t.TempDir()
	migrationsDir := filepath.Join("embed", "migrations")
	err := copyDirectory(t, "examples/sql-migrations", filepath.Join(dir, migrationsDir))
	check.NoError(t, err)
	// Create a filesystem from the temp dir
	filesystem := os.DirFS(dir)
	// Open a sqlite3 database in the temp dir
	db, err := sql.Open("sqlite", filepath.Join(dir, "test.db"))
	check.NoError(t, err)
	t.Cleanup(func() {
		check.NoError(t, db.Close())
	})
	// Create a goose provider
	opt := goose.DefaultOptions().
		SetDir(migrationsDir).
		SetFilesystem(filesystem)
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, opt)
	check.NoError(t, err)
	check.Number(t, len(provider.ListMigrations()), 3)

	version, err := provider.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, version, 0)

	_, err = provider.Up(ctx)
	check.NoError(t, err)

	version, err = provider.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, version, 3)

	_, err = provider.DownTo(ctx, 0)
	check.NoError(t, err)

	version, err = provider.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, version, 0)
}

func runGoose(t *testing.T, params ...string) string {
	t.Helper()
	dir, err := os.Getwd()
	check.NoError(t, err)
	cmdPath := filepath.Join(dir, binName)
	cmd := exec.Command(cmdPath, params...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(out))
	}
	check.NoError(t, err)
	return string(out)
}

func newDBString(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	name := "test-" + randName(6) + ".db"
	return fmt.Sprintf("sqlite:%s", filepath.Join(dir, name))
}

func randName(n int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	bytes := make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}

func countSQLFiles(t *testing.T, dir string) int {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	check.NoError(t, err)
	return len(files)
}

func copyDirectory(t *testing.T, src, dest string) error {
	t.Helper()
	entries, err := os.ReadDir(src)
	check.NoError(t, err)
	err = os.MkdirAll(dest, 0755)
	check.NoError(t, err)
	for _, file := range entries {
		if file.IsDir() {
			return fmt.Errorf("failed to copy directory. Expecting files only: %s", src)
		}
		copyFile(
			t,
			filepath.Join(src, file.Name()),
			filepath.Join(dest, file.Name()),
		)
	}
	return nil
}

func copyFile(t *testing.T, src, dest string) error {
	t.Helper()
	data, err := os.ReadFile(src)
	check.NoError(t, err)
	return os.WriteFile(dest, []byte(data), 0644)
}
