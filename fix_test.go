package goose

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

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
