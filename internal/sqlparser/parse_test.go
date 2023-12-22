package sqlparser_test

import (
	"errors"
	"os"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3/internal/check"
	"github.com/pressly/goose/v3/internal/sqlparser"
)

func TestParseAllFromFS(t *testing.T) {
	t.Parallel()
	t.Run("file_not_exist", func(t *testing.T) {
		mapFS := fstest.MapFS{}
		_, err := sqlparser.ParseAllFromFS(mapFS, "001_foo.sql", false)
		check.HasError(t, err)
		check.Bool(t, errors.Is(err, os.ErrNotExist), true)
	})
	t.Run("empty_file", func(t *testing.T) {
		mapFS := fstest.MapFS{
			"001_foo.sql": &fstest.MapFile{},
		}
		_, err := sqlparser.ParseAllFromFS(mapFS, "001_foo.sql", false)
		check.HasError(t, err)
		check.Contains(t, err.Error(), "failed to parse migration")
		check.Contains(t, err.Error(), "must start with '-- +goose Up' annotation")
	})
	t.Run("all_statements", func(t *testing.T) {
		mapFS := fstest.MapFS{
			"001_foo.sql": newFile(`
-- +goose Up
`),
			"002_bar.sql": newFile(`
-- +goose Up
-- +goose Down
`),
			"003_baz.sql": newFile(`
-- +goose Up
CREATE TABLE foo (id int);
CREATE TABLE bar (id int);

-- +goose Down
DROP TABLE bar;
`),
			"004_qux.sql": newFile(`
-- +goose NO TRANSACTION
-- +goose Up
CREATE TABLE foo (id int);
-- +goose Down
DROP TABLE foo;
`),
		}
		parsedSQL, err := sqlparser.ParseAllFromFS(mapFS, "001_foo.sql", false)
		check.NoError(t, err)
		assertParsedSQL(t, parsedSQL, true, 0, 0)
		parsedSQL, err = sqlparser.ParseAllFromFS(mapFS, "002_bar.sql", false)
		check.NoError(t, err)
		assertParsedSQL(t, parsedSQL, true, 0, 0)
		parsedSQL, err = sqlparser.ParseAllFromFS(mapFS, "003_baz.sql", false)
		check.NoError(t, err)
		assertParsedSQL(t, parsedSQL, true, 2, 1)
		parsedSQL, err = sqlparser.ParseAllFromFS(mapFS, "004_qux.sql", false)
		check.NoError(t, err)
		assertParsedSQL(t, parsedSQL, false, 1, 1)
	})
}

func assertParsedSQL(t *testing.T, got *sqlparser.ParsedSQL, useTx bool, up, down int) {
	t.Helper()
	check.Bool(t, got != nil, true)
	check.Equal(t, len(got.Up), up)
	check.Equal(t, len(got.Down), down)
	check.Equal(t, got.UseTx, useTx)
}

func newFile(data string) *fstest.MapFile {
	return &fstest.MapFile{
		Data: []byte(data),
	}
}
