package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

const (
	version = "devel"
)

func TestRun(t *testing.T) {
	t.Run("version", func(t *testing.T) {
		stdout, stderr, err := runCommand("--version")
		require.NoError(t, err)
		assert.Empty(t, stderr)
		assert.Equal(t, stdout, "goose version: "+version+"\n")
	})
	t.Run("with_filesystem", func(t *testing.T) {
		fsys := fstest.MapFS{
			"migrations/001_foo.sql": {Data: []byte(`-- +goose up`)},
		}
		command := "status --dir=migrations --dbstring=sqlite3://:memory: --json"
		buf := bytes.NewBuffer(nil)
		err := Run(context.Background(), strings.Split(command, " "), WithFilesystem(fsys.Sub), WithStdout(buf))
		require.NoError(t, err)
		var status migrationsStatus
		err = json.Unmarshal(buf.Bytes(), &status)
		require.NoError(t, err)
		require.Len(t, status.Migrations, 1)
		assert.True(t, status.HasPending)
		assert.Equal(t, "001_foo.sql", status.Migrations[0].Source.Path)
		assert.Equal(t, "pending", status.Migrations[0].State)
		assert.Equal(t, "", status.Migrations[0].AppliedAt)
	})
}

func runCommand(args ...string) (string, string, error) {
	stdout, stderr := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	err := Run(
		context.Background(),
		args,
		WithStdout(stdout),
		WithStderr(stderr),
		WithVersion(version),
	)
	return stdout.String(), stderr.String(), err
}
