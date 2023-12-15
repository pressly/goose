package gomigrations_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"

	_ "github.com/pressly/goose/v3/tests/gomigrations/success/testdata"
	_ "modernc.org/sqlite"
)

func TestGoMigrationByOne(t *testing.T) {
	t.Parallel()

	check.NoError(t, goose.SetDialect("sqlite3"))
	db, err := sql.Open("sqlite", ":memory:")
	check.NoError(t, err)
	dir := "testdata"
	files, err := filepath.Glob(dir + "/*.go")
	check.NoError(t, err)

	upByOne := func(t *testing.T) int64 {
		err = goose.UpByOne(db, dir)
		t.Logf("err: %v %s", err, dir)
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

	tables, err := ListTables(db)
	check.NoError(t, err)
	check.Equal(t, tables, []string{
		"alpha",
		"bravo",
		"charlie",
		"delta",
		"echo",
		"foxtrot",
		"golf",
		"goose_db_version",
		"hotel",
		"sqlite_sequence",
	})

	// Migrate all files down-by-one.
	for i := len(files) - 1; i >= 0; i-- {
		check.Number(t, downByOne(t), i)
	}
	version, err = goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, version, 0)

	tables, err = ListTables(db)
	check.NoError(t, err)
	check.Equal(t, tables, []string{
		"goose_db_version",
		"sqlite_sequence",
	})
}

func ListTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tables, nil
}
