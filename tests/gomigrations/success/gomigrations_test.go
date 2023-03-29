package gomigrations

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v4"
	"github.com/pressly/goose/v4/internal/check"
	"github.com/pressly/goose/v4/internal/testdb"

	_ "github.com/pressly/goose/v4/tests/gomigrations/success/testdata"
)

func TestGoMigrationByOne(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	db, cleanup, err := testdb.NewPostgres()
	check.NoError(t, err)
	t.Cleanup(cleanup)

	options := goose.DefaultOptions().
		SetDir("testdata").
		SetVerbose(testing.Verbose())
	p, err := goose.NewProvider(goose.DialectPostgres, db, options)
	check.NoError(t, err)

	// TODO(mf): add a tests to detect 1 open connection against *sql.DB. This deadlocks
	// so we should handle this gracefully.

	dir := "testdata"
	files, err := filepath.Glob(dir + "/*.go")
	check.NoError(t, err)

	upByOne := func(t *testing.T) int64 {
		_, err = p.UpByOne(ctx)
		check.NoError(t, err)
		version, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		return version
	}
	downByOne := func(t *testing.T) int64 {
		_, err = p.Down(ctx)
		check.NoError(t, err)
		version, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		return version
	}
	// Migrate all files up-by-one.
	for i := 1; i <= len(files); i++ {
		check.Number(t, upByOne(t), i)
	}
	version, err := p.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, version, len(files))

	// Migrate all files down-by-one.
	for i := len(files) - 1; i >= 0; i-- {
		check.Number(t, downByOne(t), i)
	}
	version, err = p.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, version, 0)
}
