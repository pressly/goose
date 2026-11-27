package goose

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// A valid timestamp at the Unix epoch must not disappear between the sequential
// and timestamped migration classifications used by Fix.
func TestFixTimestampAtUnixEpoch(t *testing.T) {
	dir := t.TempDir()
	files := []struct {
		original string
		fixed    string
		content  string
	}{
		{"19700101000000_epoch.sql", "00001_epoch.sql", "SELECT 'epoch';"},
		{"19700101000001_after.sql", "00002_after.sql", "SELECT 'after';"},
	}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(dir, file.original), []byte(file.content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := Fix(dir); err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		data, err := os.ReadFile(filepath.Join(dir, file.fixed))
		if err != nil {
			t.Errorf("migration %s was not renumbered to %s: %v", file.original, file.fixed, err)
			continue
		}
		if string(data) != file.content {
			t.Errorf("migration %s contents changed: %q", file.fixed, data)
		}
		if _, err := os.Stat(filepath.Join(dir, file.original)); !os.IsNotExist(err) {
			t.Errorf("original timestamped migration %s remains: %v", file.original, err)
		}
	}
}

func TestFix(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skip long running test")
	}

	dir := t.TempDir()
	defer os.Remove("./bin/fix-goose") // clean up

	commands := []string{
		"go build -o ./bin/fix-goose ./cmd/goose",
		fmt.Sprintf("./bin/fix-goose -dir=%s create create_table", dir),
		fmt.Sprintf("./bin/fix-goose -dir=%s create add_users", dir),
		fmt.Sprintf("./bin/fix-goose -dir=%s create add_indices", dir),
		fmt.Sprintf("./bin/fix-goose -dir=%s create update_users", dir),
		fmt.Sprintf("./bin/fix-goose -dir=%s fix", dir),
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

	// add more migrations and then fix it
	commands = []string{
		fmt.Sprintf("./bin/fix-goose -dir=%s create remove_column", dir),
		fmt.Sprintf("./bin/fix-goose -dir=%s create create_books_table", dir),
		fmt.Sprintf("./bin/fix-goose -dir=%s fix", dir),
	}

	for _, cmd := range commands {
		args := strings.Split(cmd, " ")
		time.Sleep(1 * time.Second)
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			t.Fatalf("%s:\n%v\n\n%s", err, cmd, out)
		}
	}

	files, err = os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	// check that the files still in order
	for i, f := range files {
		expected := fmt.Sprintf("%05v", i+1)
		if !strings.HasPrefix(f.Name(), expected) {
			t.Errorf("failed to find %s prefix in %s", expected, f.Name())
		}
	}
}
