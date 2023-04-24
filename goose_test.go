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
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"

	"github.com/pressly/goose/v4"
	"github.com/pressly/goose/v4/internal/check"
	"github.com/pressly/goose/v4/internal/testdb"
	"golang.org/x/sync/errgroup"
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
		// disable all drivers except sqlite3 and postgres
		"-tags=no_clickhouse no_mssql no_mysql no_vertica",
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
	out, err := runGoose("--version")
	check.NoError(t, err)
	check.Contains(t, out, "goose version: (devel)")
}

func TestGooseInit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dirFlag := "--dir=" + filepath.Join(dir, "migrations")
	out, err := runGoose("init", dirFlag)
	check.NoError(t, err)
	check.Contains(t, out, "00001_initial.sql")
}

func TestGooseCreate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dirFlag := "--dir=" + filepath.Join(dir, "migrations")
	out, err := runGoose("create", "-s", dirFlag, "sql", "add users table")
	check.NoError(t, err)
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
		{"version", "", "goose: version 3"},
		{"up", "", "no migrations to run"},
		{"down-to", "0", ""},
		{"version", "", "goose: version 0"},
		{"down-to", "0", "no migrations to run"},
		{"status", "", ""},
	}
	for _, tc := range tt {
		params := []string{tc.command, dirFlag, dbStringFlag}
		params = append(params, strings.Split(tc.args, " ")...)
		got, err := runGoose(params...)
		check.NoError(t, err)
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

func TestBinaryLockMode(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skip long running test")
	}
	const (
		maxWorkers = 5
	)
	port := randomPort()
	_, cleanup, err := testdb.NewPostgres(testdb.WithBindPort(port))
	check.NoError(t, err)
	t.Cleanup(cleanup)

	migrationsDir := "testdata/binary-lock-mode"
	total := countSQLFiles(t, migrationsDir)
	check.Number(t, total, 3)

	dirFlag := "--dir=" + migrationsDir
	dbStringFlag := "--dbstring=" + fmt.Sprintf("postgres://postgres:password1@localhost:%d/testdb?sslmode=disable", port)

	// Due to the way the migrations are written, they will fail if run concurrently. Try setting
	// this to an empty string to see the failures.
	lockModeFlag := "--lock-mode=" + "advisory-session"

	var g errgroup.Group

	for i := 0; i < maxWorkers; i++ {
		g.Go(func() error {
			_, err := runGoose("up", dirFlag, dbStringFlag, lockModeFlag)
			return err
		})
	}
	err = g.Wait()
	check.NoError(t, err)

	out, err := runGoose("version", dirFlag, dbStringFlag, lockModeFlag)
	check.NoError(t, err)
	check.Contains(t, out, "goose: version "+strconv.Itoa(total))

	for i := 0; i < maxWorkers; i++ {
		g.Go(func() error {
			_, err := runGoose("down-to", dirFlag, dbStringFlag, lockModeFlag, "0")
			return err
		})
	}
	err = g.Wait()
	check.NoError(t, err)

	out, err = runGoose("version", dirFlag, dbStringFlag, lockModeFlag)
	check.NoError(t, err)
	check.Contains(t, out, "goose: version 0")
}

func TestParallelStatus(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skip long running test")
	}
	const (
		maxWorkers = 5
	)
	port := randomPort()
	_, cleanup, err := testdb.NewPostgres(testdb.WithBindPort(port))
	check.NoError(t, err)
	t.Cleanup(cleanup)

	migrationsDir := "testdata/binary-lock-mode"
	total := countSQLFiles(t, migrationsDir)
	check.Number(t, total, 3)

	dirFlag := "--dir=" + migrationsDir
	dbStringFlag := "--dbstring=" + fmt.Sprintf("postgres://postgres:password1@localhost:%d/testdb?sslmode=disable", port)

	// In this test, we don't apply any migrations, but we do run status in parallel. Without
	// successfully acquiring a lock, the status command will fail to create the goose migrations
	// table.
	//
	// duplicate key value violates unique constraint "pg_class_relname_nsp_index" (SQLSTATE 23505)
	//
	// The "pg_class_relname_nsp_index" constraint specifically pertains to the "pg_class" system
	// catalog table, which stores information about tables in a PostgreSQL database.
	lockModeFlag := "--lock-mode=" + "advisory-session"

	var g errgroup.Group

	var output []string
	var mu sync.Mutex
	for i := 0; i < maxWorkers; i++ {
		g.Go(func() error {
			out, err := runGoose("version", dirFlag, dbStringFlag, lockModeFlag)
			mu.Lock()
			defer mu.Unlock()
			output = append(output, out)
			return err
		})
	}
	err = g.Wait()
	check.NoError(t, err)
	for _, out := range output {
		check.Contains(t, out, "goose: version 0")
	}
}

func TestEmbedBinary(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	/*
		To avoid accidental changes to the embedded migrations, we copy them to a temp dir.

		In a real application, you would use the migrations embedded in your binary. For example:

		//go:embed examples/sql-migrations/*.sql var migrations embed.FS

		opt := goose.DefaultOptions(). SetDir("examples/sql-migrations"). SetFilesystem(migrations)
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

func copyFile(t *testing.T, src, dest string) {
	t.Helper()
	data, err := os.ReadFile(src)
	check.NoError(t, err)
	err = os.WriteFile(dest, []byte(data), 0644)
	check.NoError(t, err)
}

func randomPort() int {
	rand.Seed(time.Now().UnixNano())
	min, max := 32768, 61000
	// Generate a random number within the range [min, max]
	return rand.Intn(max-min+1) + min
}
