package testdata

import (
	"bytes"
	"os"
	"testing"

	"github.com/pressly/goose/v3/internal/gooseutil"
	"github.com/stretchr/testify/require"
)

func TestEmbededMigrations(t *testing.T) {
	t.Parallel()

	files, err := EmbedMigrations.ReadDir(".")
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, "migrations", files[0].Name())

	files, err = EmbedMigrations.ReadDir("migrations")
	require.NoError(t, err)
	got := make([]string, 0, len(files))
	for _, file := range files {
		got = append(got, file.Name())
	}
	require.ElementsMatch(t, []string{"postgres"}, got)

	t.Run("postgres", func(t *testing.T) {
		expected, err := os.ReadFile("migrations/postgres.sha256")
		require.NoError(t, err)
		expected = bytes.TrimSpace(expected)
		files, err := EmbedMigrations.ReadDir("migrations/postgres")
		require.NoError(t, err)
		require.Len(t, files, 5)

		digest, err := gooseutil.Digest(EmbedMigrations, "migrations/postgres")
		require.NoError(t, err)

		require.Equal(t, string(expected), digest)
	})

}
