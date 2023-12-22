package goose

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
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
