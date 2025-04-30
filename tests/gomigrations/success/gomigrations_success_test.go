package gomigrations_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	_ "github.com/pressly/goose/v3/tests/gomigrations/success/testdata"
	_ "modernc.org/sqlite"
)

func TestGoMigrationByOne(t *testing.T) {
	t.Parallel()

	require.NoError(t, goose.SetDialect("sqlite3"))
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	dir := "testdata"
	files, err := filepath.Glob(dir + "/*.go")
	require.NoError(t, err)

	upByOne := func(t *testing.T) int64 {
		t.Helper()
		err = goose.UpByOne(db, dir)
		t.Logf("err: %v %s", err, dir)
		require.NoError(t, err)
		version, err := goose.GetDBVersion(db)
		require.NoError(t, err)
		return version
	}
	downByOne := func(t *testing.T) int64 {
		t.Helper()
		err = goose.Down(db, dir)
		require.NoError(t, err)
		version, err := goose.GetDBVersion(db)
		require.NoError(t, err)
		return version
	}
	// Migrate all files up-by-one.
	for i := 1; i <= len(files); i++ {
		require.EqualValues(t, upByOne(t), i)
	}
	version, err := goose.GetDBVersion(db)
	require.NoError(t, err)
	require.Len(t, files, int(version))

	tables, err := ListTables(db)
	require.NoError(t, err)
	require.Equal(t,
		[]string{
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
		},
		tables,
	)

	// Migrate all files down-by-one.
	for i := len(files) - 1; i >= 0; i-- {
		require.EqualValues(t, downByOne(t), i)
	}
	version, err = goose.GetDBVersion(db)
	require.NoError(t, err)
	require.EqualValues(t, 0, version)

	tables, err = ListTables(db)
	require.NoError(t, err)
	require.Equal(t,
		[]string{
			"goose_db_version",
			"sqlite_sequence",
		},
		tables,
	)
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
