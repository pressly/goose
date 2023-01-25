package gomigrations

import (
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
	"github.com/pressly/goose/v3/internal/testdb"

	_ "github.com/pressly/goose/v3/tests/gomigrations/success/testdata"
)

func TestGoMigrationByOne(t *testing.T) {
	db, cleanup, err := testdb.NewPostgres()
	check.NoError(t, err)
	t.Cleanup(cleanup)

	dir := "testdata"
	files, err := filepath.Glob(dir + "/*.go")
	check.NoError(t, err)

	upByOne := func(t *testing.T) int64 {
		err = goose.UpByOne(db, dir)
		check.NoError(t, err)
		version, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		return version
	}
	downByOne := func(t *testing.T) int64 {
		err = goose.Down(db, dir)
		check.NoError(t, err)
		version, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		return version
	}
	// Migrate all files up-by-one.
	for i := 1; i <= len(files); i++ {
		check.Number(t, upByOne(t), i)
	}
	version, err := goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, version, len(files))

	// Migrate all files down-by-one.
	for i := len(files) - 1; i >= 0; i-- {
		check.Number(t, downByOne(t), i)
	}
	version, err = goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, version, 0)
}
