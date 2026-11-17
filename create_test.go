package goose

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSequential(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skip long running test")
	}

	dir := t.TempDir()
	defer os.Remove("./bin/create-goose") // clean up

	commands := []string{
		"go build -o ./bin/create-goose ./cmd/goose",
		fmt.Sprintf("./bin/create-goose -s -dir=%s create create_table", dir),
		fmt.Sprintf("./bin/create-goose -s -dir=%s create add_users", dir),
		fmt.Sprintf("./bin/create-goose -s -dir=%s create add_indices", dir),
		fmt.Sprintf("./bin/create-goose -s -dir=%s create update_users", dir),
	}

	for _, cmd := range commands {
		args := strings.Split(cmd, " ")
		time.Sleep(1 * time.Second)
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s:\n%v\n\n%s", err, cmd, out)
		}
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	// check that the files are in order
	for i, f := range files {
		expected := fmt.Sprintf("%05v", i+1)
		if !strings.HasPrefix(f.Name(), expected) {
			t.Errorf("failed to find %s prefix in %s", expected, f.Name())
		}
	}
}

func TestCreateDuplicateFile(t *testing.T) {
	prev := timestampFormat
	timestampFormat = "STATIC"
	t.Cleanup(func() { timestampFormat = prev })

	dir := t.TempDir()

	if err := Create(nil, dir, "add_users", "sql"); err != nil {
		t.Fatalf("first create should succeed: %v", err)
	}
	err := Create(nil, dir, "add_users", "sql")
	if err == nil {
		t.Fatal("second create should have failed, got nil")
	}
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("want os.ErrExist, got %v", err)
	}
}

func TestCreateTimestampSameSecondCollision(t *testing.T) {
	frozen := time.Date(2026, 9, 3, 21, 19, 32, 0, time.UTC)
	prevNow := timeNow
	timeNow = func() time.Time { return frozen }
	t.Cleanup(func() { timeNow = prevNow })

	dir := t.TempDir()
	if err := Create(nil, dir, "alpha", "sql"); err != nil {
		t.Fatalf("first create should succeed: %v", err)
	}
	if err := Create(nil, dir, "beta", "sql"); err != nil {
		t.Fatalf("second create should bump version, got %v", err)
	}

	got := migrationFilenames(t, dir)
	want := []string{
		"20260903211932_alpha.sql",
		"20260903211933_beta.sql",
	}
	if len(got) != len(want) {
		t.Fatalf("got files %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got files %v, want %v", got, want)
		}
	}

	assertUniqueVersions(t, dir)
}

func TestCreateTimestampSameSecondCollisionLaterNameFirst(t *testing.T) {
	// Creating the lexicographically later name first must still bump the
	// second file. The later file already owns the version (it is non-empty).
	frozen := time.Date(2026, 9, 3, 21, 19, 32, 0, time.UTC)
	prevNow := timeNow
	timeNow = func() time.Time { return frozen }
	t.Cleanup(func() { timeNow = prevNow })

	dir := t.TempDir()
	if err := Create(nil, dir, "beta", "sql"); err != nil {
		t.Fatalf("first create should succeed: %v", err)
	}
	if err := Create(nil, dir, "alpha", "sql"); err != nil {
		t.Fatalf("second create should bump version, got %v", err)
	}

	got := migrationFilenames(t, dir)
	want := []string{
		"20260903211932_beta.sql",
		"20260903211933_alpha.sql",
	}
	if len(got) != len(want) {
		t.Fatalf("got files %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got files %v, want %v", got, want)
		}
	}
	assertUniqueVersions(t, dir)
}

func TestCreateSequentialCollisionRetry(t *testing.T) {
	prev := sequential
	SetSequential(true)
	t.Cleanup(func() { SetSequential(prev) })

	dir := t.TempDir()
	seed := filepath.Join(dir, "00001_existing.sql")
	if err := os.WriteFile(seed, []byte("-- seed\n"), 0o666); err != nil {
		t.Fatal(err)
	}

	if err := Create(nil, dir, "beta", "sql"); err != nil {
		t.Fatalf("create should skip taken version, got %v", err)
	}

	got := migrationFilenames(t, dir)
	want := []string{"00001_existing.sql", "00002_beta.sql"}
	if len(got) != len(want) {
		t.Fatalf("got files %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got files %v, want %v", got, want)
		}
	}
	assertUniqueVersions(t, dir)
}

func TestCreateSequentialExistingDuplicates(t *testing.T) {
	prev := sequential
	SetSequential(true)
	t.Cleanup(func() { SetSequential(prev) })

	dir := t.TempDir()
	for _, name := range []string{"00001_a.sql", "00001_b.sql"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("-- seed\n"), 0o666); err != nil {
			t.Fatal(err)
		}
	}

	if err := Create(nil, dir, "c", "sql"); err != nil {
		t.Fatalf("create should not panic on existing duplicate versions, got %v", err)
	}

	found := false
	for _, name := range migrationFilenames(t, dir) {
		if name == "00002_c.sql" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 00002_c.sql among %v", migrationFilenames(t, dir))
	}

	// The new file must not reuse version 1.
	v, err := NumericComponent("00002_c.sql")
	if err != nil {
		t.Fatal(err)
	}
	if v != 2 {
		t.Fatalf("got version %d, want 2", v)
	}
}

func TestCreateSequentialConcurrent(t *testing.T) {
	prev := sequential
	SetSequential(true)
	t.Cleanup(func() { SetSequential(prev) })

	dir := t.TempDir()
	names := []string{"a", "b", "c", "d", "e", "f", "g", "h"}

	errCh := make(chan error, len(names))
	var wg sync.WaitGroup
	for _, name := range names {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- Create(nil, dir, name, "sql")
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent create failed: %v", err)
		}
	}

	got := migrationFilenames(t, dir)
	if len(got) != len(names) {
		t.Fatalf("got %d files %v, want %d", len(got), got, len(names))
	}
	assertUniqueVersions(t, dir)
}

func migrationFilenames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

func assertUniqueVersions(t *testing.T, dir string) {
	t.Helper()
	seen := make(map[int64]string)
	for _, name := range migrationFilenames(t, dir) {
		v, err := NumericComponent(name)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		if existing, ok := seen[v]; ok {
			t.Fatalf("duplicate version %d: %s and %s", v, existing, name)
		}
		seen[v] = name
	}
}
