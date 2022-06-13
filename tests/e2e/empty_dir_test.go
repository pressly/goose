package e2e

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
)

func TestEmptyMigrationsDirectory(t *testing.T) {
	t.Parallel()

	migrationsDir, err := ioutil.TempDir(os.TempDir(), "migrations")
	if err != nil {
		check.NoError(t, err)
	}
	defer os.RemoveAll(migrationsDir)

	_, err = newDockerDB(t)
	check.NoError(t, err)
	goose.SetDialect(*dialect)

	_, err = goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	check.Contains(t, err.Error(), "directory does not contain any kind of migration files")
}
