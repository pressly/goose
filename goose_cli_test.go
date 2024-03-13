package goose_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/pressly/goose/v3/internal/check"
	_ "modernc.org/sqlite"
)

const (
	// gooseTestBinaryVersion is utilized in conjunction with a linker variable to set the version
	// of a binary created solely for testing purposes. It is used to test the --version flag.
	gooseTestBinaryVersion = "v0.0.0"
)

func TestFullBinary(t *testing.T) {
	t.Parallel()
	cli := buildGooseCLI(t, false)
	out, err := cli.run("--version")
	check.NoError(t, err)
	check.Equal(t, out, "goose version: "+gooseTestBinaryVersion+"\n")
}

func TestLiteBinary(t *testing.T) {
	t.Parallel()
	cli := buildGooseCLI(t, true)

	t.Run("binary_version", func(t *testing.T) {
		t.Parallel()
		out, err := cli.run("--version")
		check.NoError(t, err)
		check.Equal(t, out, "goose version: "+gooseTestBinaryVersion+"\n")
	})
	t.Run("default_binary", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		total := countSQLFiles(t, "testdata/migrations")
		commands := []struct {
			cmd string
			out string
		}{
			{"up", "goose: successfully migrated database to version: " + strconv.Itoa(total)},
			{"version", "goose: version " + strconv.Itoa(total)},
			{"down", "OK"},
			{"version", "goose: version " + strconv.Itoa(total-1)},
			{"status", ""},
			{"reset", "OK"},
			{"version", "goose: version 0"},
		}
		for _, c := range commands {
			out, err := cli.run("-dir=testdata/migrations", "sqlite3", filepath.Join(dir, "sql.db"), c.cmd)
			check.NoError(t, err)
			check.Contains(t, out, c.out)
		}
	})
	t.Run("gh_issue_532", func(t *testing.T) {
		// https://github.com/pressly/goose/issues/532
		t.Parallel()
		dir := t.TempDir()
		total := countSQLFiles(t, "testdata/migrations")
		_, err := cli.run("-dir=testdata/migrations", "sqlite3", filepath.Join(dir, "sql.db"), "up")
		check.NoError(t, err)
		out, err := cli.run("-dir=testdata/migrations", "sqlite3", filepath.Join(dir, "sql.db"), "up")
		check.NoError(t, err)
		check.Contains(t, out, "goose: no migrations to run. current version: "+strconv.Itoa(total))
		out, err = cli.run("-dir=testdata/migrations", "sqlite3", filepath.Join(dir, "sql.db"), "version")
		check.NoError(t, err)
		check.Contains(t, out, "goose: version "+strconv.Itoa(total))
	})
	t.Run("gh_issue_293", func(t *testing.T) {
		// https://github.com/pressly/goose/issues/293
		t.Parallel()
		dir := t.TempDir()
		total := countSQLFiles(t, "testdata/migrations")
		commands := []struct {
			cmd string
			out string
		}{
			{"up", "goose: successfully migrated database to version: " + strconv.Itoa(total)},
			{"version", "goose: version " + strconv.Itoa(total)},
			{"down", "OK"},
			{"down", "OK"},
			{"version", "goose: version " + strconv.Itoa(total-2)},
			{"up", "goose: successfully migrated database to version: " + strconv.Itoa(total)},
			{"status", ""},
		}
		for _, c := range commands {
			out, err := cli.run("-dir=testdata/migrations", "sqlite3", filepath.Join(dir, "sql.db"), c.cmd)
			check.NoError(t, err)
			check.Contains(t, out, c.out)
		}
	})
	t.Run("gh_issue_336", func(t *testing.T) {
		// https://github.com/pressly/goose/issues/336
		t.Parallel()
		dir := t.TempDir()
		_, err := cli.run("-dir="+dir, "sqlite3", filepath.Join(dir, "sql.db"), "up")
		check.HasError(t, err)
		check.Contains(t, err.Error(), "goose run: no migration files found")
	})
	t.Run("create_and_fix", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		createEmptyFile(t, dir, "00001_alpha.sql")
		createEmptyFile(t, dir, "00003_bravo.sql")
		createEmptyFile(t, dir, "20230826163141_charlie.sql")
		createEmptyFile(t, dir, "20230826163151_delta.go")
		total, err := os.ReadDir(dir)
		check.NoError(t, err)
		check.Number(t, len(total), 4)
		migrationFiles := []struct {
			name     string
			fileType string
		}{
			{"echo", "sql"},
			{"foxtrot", "go"},
			{"golf", ""},
		}
		for i, f := range migrationFiles {
			args := []string{"-dir=" + dir, "create", f.name}
			if f.fileType != "" {
				args = append(args, f.fileType)
			}
			out, err := cli.run(args...)
			check.NoError(t, err)
			check.Contains(t, out, "Created new file")
			// ensure different timestamps, granularity is 1 second
			if i < len(migrationFiles)-1 {
				time.Sleep(1100 * time.Millisecond)
			}
		}
		total, err = os.ReadDir(dir)
		check.NoError(t, err)
		check.Number(t, len(total), 7)
		out, err := cli.run("-dir="+dir, "fix")
		check.NoError(t, err)
		check.Contains(t, out, "RENAMED")
		files, err := os.ReadDir(dir)
		check.NoError(t, err)
		check.Number(t, len(files), 7)
		expected := []string{
			"00001_alpha.sql",
			"00003_bravo.sql",
			"00004_charlie.sql",
			"00005_delta.go",
			"00006_echo.sql",
			"00007_foxtrot.go",
			"00008_golf.go",
		}
		for i, f := range files {
			check.Equal(t, f.Name(), expected[i])
		}
	})
}

type gooseBinary struct {
	binaryPath string
}

func (g gooseBinary) run(params ...string) (string, error) {
	cmd := exec.Command(g.binaryPath, params...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run goose command: %v\nout: %v", err, string(out))
	}
	return string(out), nil
}

// buildGooseCLI builds goose test binary, which is used for testing goose CLI. It is built with all
// drivers enabled, unless lite is true, in which case all drivers are disabled except sqlite3
func buildGooseCLI(t *testing.T, lite bool) gooseBinary {
	binName := "goose-test"
	dir := t.TempDir()
	output := filepath.Join(dir, binName)
	// usage: go build [-o output] [build flags] [packages]
	args := []string{
		"build",
		"-o", output,
		"-ldflags=-s -w -X main.version=" + gooseTestBinaryVersion,
	}
	if lite {
		args = append(args, "-tags=no_clickhouse no_mssql no_mysql no_vertica no_postgres")
	}
	args = append(args, "./cmd/goose")
	build := exec.Command("go", args...)
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build %s binary: %v: %s", binName, err, string(out))
	}
	return gooseBinary{
		binaryPath: output,
	}
}

func countSQLFiles(t *testing.T, dir string) int {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	check.NoError(t, err)
	return len(files)
}

func createEmptyFile(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	check.NoError(t, err)
	defer f.Close()
}
