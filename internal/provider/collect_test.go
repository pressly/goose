package provider

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3/internal/check"
)

func TestCollectFileSources(t *testing.T) {
	t.Parallel()
	t.Run("nil", func(t *testing.T) {
		sources, err := collectFileSources(nil, false, nil)
		check.NoError(t, err)
		check.Bool(t, sources != nil, true)
		check.Number(t, len(sources.goSources), 0)
		check.Number(t, len(sources.sqlSources), 0)
	})
	t.Run("empty", func(t *testing.T) {
		sources, err := collectFileSources(fstest.MapFS{}, false, nil)
		check.NoError(t, err)
		check.Number(t, len(sources.goSources), 0)
		check.Number(t, len(sources.sqlSources), 0)
		check.Bool(t, sources != nil, true)
	})
	t.Run("incorrect_fsys", func(t *testing.T) {
		mapFS := fstest.MapFS{
			"00000_foo.sql": sqlMapFile,
		}
		// strict disable - should not error
		sources, err := collectFileSources(mapFS, false, nil)
		check.NoError(t, err)
		check.Number(t, len(sources.goSources), 0)
		check.Number(t, len(sources.sqlSources), 0)
		// strict enabled - should error
		_, err = collectFileSources(mapFS, true, nil)
		check.HasError(t, err)
		check.Contains(t, err.Error(), "migration version must be greater than zero")
	})
	t.Run("collect", func(t *testing.T) {
		fsys, err := fs.Sub(newSQLOnlyFS(), "migrations")
		check.NoError(t, err)
		sources, err := collectFileSources(fsys, false, nil)
		check.NoError(t, err)
		check.Number(t, len(sources.sqlSources), 4)
		check.Number(t, len(sources.goSources), 0)
		expected := fileSources{
			sqlSources: []Source{
				{Fullpath: "00001_foo.sql", Version: 1},
				{Fullpath: "00002_bar.sql", Version: 2},
				{Fullpath: "00003_baz.sql", Version: 3},
				{Fullpath: "00110_qux.sql", Version: 110},
			},
		}
		for i := 0; i < len(sources.sqlSources); i++ {
			check.Equal(t, sources.sqlSources[i], expected.sqlSources[i])
		}
	})
	t.Run("excludes", func(t *testing.T) {
		fsys, err := fs.Sub(newSQLOnlyFS(), "migrations")
		check.NoError(t, err)
		sources, err := collectFileSources(
			fsys,
			false,
			// exclude 2 files explicitly
			map[string]bool{
				"00002_bar.sql": true,
				"00110_qux.sql": true,
			},
		)
		check.NoError(t, err)
		check.Number(t, len(sources.sqlSources), 2)
		check.Number(t, len(sources.goSources), 0)
		expected := fileSources{
			sqlSources: []Source{
				{Fullpath: "00001_foo.sql", Version: 1},
				{Fullpath: "00003_baz.sql", Version: 3},
			},
		}
		for i := 0; i < len(sources.sqlSources); i++ {
			check.Equal(t, sources.sqlSources[i], expected.sqlSources[i])
		}
	})
	t.Run("strict", func(t *testing.T) {
		mapFS := newSQLOnlyFS()
		// Add a file with no version number
		mapFS["migrations/not_valid.sql"] = &fstest.MapFile{Data: []byte("invalid")}
		fsys, err := fs.Sub(mapFS, "migrations")
		check.NoError(t, err)
		_, err = collectFileSources(fsys, true, nil)
		check.HasError(t, err)
		check.Contains(t, err.Error(), `failed to parse numeric component from "not_valid.sql"`)
	})
	t.Run("skip_go_test_files", func(t *testing.T) {
		mapFS := fstest.MapFS{
			"1_foo.sql":     sqlMapFile,
			"2_bar.sql":     sqlMapFile,
			"3_baz.sql":     sqlMapFile,
			"4_qux.sql":     sqlMapFile,
			"5_foo_test.go": {Data: []byte(`package goose_test`)},
		}
		sources, err := collectFileSources(mapFS, false, nil)
		check.NoError(t, err)
		check.Number(t, len(sources.sqlSources), 4)
		check.Number(t, len(sources.goSources), 0)
	})
	t.Run("skip_random_files", func(t *testing.T) {
		mapFS := fstest.MapFS{
			"1_foo.sql":                sqlMapFile,
			"4_something.go":           {Data: []byte(`package goose`)},
			"5_qux.sql":                sqlMapFile,
			"README.md":                {Data: []byte(`# README`)},
			"LICENSE":                  {Data: []byte(`MIT`)},
			"no_a_real_migration.sql":  {Data: []byte(`SELECT 1;`)},
			"some/other/dir/2_foo.sql": {Data: []byte(`SELECT 1;`)},
		}
		sources, err := collectFileSources(mapFS, false, nil)
		check.NoError(t, err)
		check.Number(t, len(sources.sqlSources), 2)
		check.Number(t, len(sources.goSources), 1)
		// 1
		check.Equal(t, sources.sqlSources[0].Fullpath, "1_foo.sql")
		check.Equal(t, sources.sqlSources[0].Version, int64(1))
		// 2
		check.Equal(t, sources.sqlSources[1].Fullpath, "5_qux.sql")
		check.Equal(t, sources.sqlSources[1].Version, int64(5))
		// 3
		check.Equal(t, sources.goSources[0].Fullpath, "4_something.go")
		check.Equal(t, sources.goSources[0].Version, int64(4))
	})
	t.Run("duplicate_versions", func(t *testing.T) {
		mapFS := fstest.MapFS{
			"001_foo.sql": sqlMapFile,
			"01_bar.sql":  sqlMapFile,
		}
		_, err := collectFileSources(mapFS, false, nil)
		check.HasError(t, err)
		check.Contains(t, err.Error(), "found duplicate migration version 1")
	})
	t.Run("dirpath", func(t *testing.T) {
		mapFS := fstest.MapFS{
			"dir1/101_a.sql": sqlMapFile,
			"dir1/102_b.sql": sqlMapFile,
			"dir1/103_c.sql": sqlMapFile,
			"dir2/201_a.sql": sqlMapFile,
			"876_a.sql":      sqlMapFile,
		}
		assertDirpath := func(dirpath string, sqlSources []Source) {
			t.Helper()
			f, err := fs.Sub(mapFS, dirpath)
			check.NoError(t, err)
			got, err := collectFileSources(f, false, nil)
			check.NoError(t, err)
			check.Number(t, len(got.sqlSources), len(sqlSources))
			check.Number(t, len(got.goSources), 0)
			for i := 0; i < len(got.sqlSources); i++ {
				check.Equal(t, got.sqlSources[i], sqlSources[i])
			}
		}
		assertDirpath(".", []Source{
			{Fullpath: "876_a.sql", Version: 876},
		})
		assertDirpath("dir1", []Source{
			{Fullpath: "101_a.sql", Version: 101},
			{Fullpath: "102_b.sql", Version: 102},
			{Fullpath: "103_c.sql", Version: 103},
		})
		assertDirpath("dir2", []Source{{Fullpath: "201_a.sql", Version: 201}})
		assertDirpath("dir3", nil)
	})
}

func newSQLOnlyFS() fstest.MapFS {
	return fstest.MapFS{
		"migrations/00001_foo.sql": sqlMapFile,
		"migrations/00002_bar.sql": sqlMapFile,
		"migrations/00003_baz.sql": sqlMapFile,
		"migrations/00110_qux.sql": sqlMapFile,
	}
}

var (
	sqlMapFile = &fstest.MapFile{Data: []byte(`-- +goose Up`)}
)
