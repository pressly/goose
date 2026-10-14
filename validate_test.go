package goose_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	goose.ResetGlobalMigrations()
	t.Cleanup(goose.ResetGlobalMigrations)

	t.Run("valid_migrations", func(t *testing.T) {
		require.NoError(t, goose.Validate("testdata/migrations"))
	})
	t.Run("run_command", func(t *testing.T) {
		// The validate command must be available to custom binaries calling the
		// goose.Run family of functions.
		require.NoError(t, goose.RunContext(context.Background(), "validate", nil, "testdata/migrations"))
	})
	t.Run("invalid_sql_migration", func(t *testing.T) {
		dir := t.TempDir()
		// Missing the required '-- +goose Up' annotation.
		writeFile(t, dir, "00001_no_annotation.sql", "SELECT 1;\n")
		err := goose.Validate(dir)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse file")
	})
	t.Run("unregistered_go_migration", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "00001_users.go", "package migrations\n")
		err := goose.Validate(dir)
		require.Error(t, err)
		require.Contains(t, err.Error(), "must be registered")
	})
	t.Run("dir_does_not_exist", func(t *testing.T) {
		err := goose.Validate(filepath.Join(t.TempDir(), "does-not-exist"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "directory does not exist")
	})
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	require.NoError(t, err)
}
