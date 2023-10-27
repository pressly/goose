package provider

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3/database"
	"github.com/pressly/goose/v3/internal/check"
)

func TestCollectFileSources(t *testing.T) {
	t.Parallel()
	t.Run("nil_fsys", func(t *testing.T) {
		sources, err := collectFileSources(nil, false, nil)
		check.NoError(t, err)
		check.Bool(t, sources != nil, true)
		check.Number(t, len(sources.goSources), 0)
		check.Number(t, len(sources.sqlSources), 0)
	})
	t.Run("empty_fsys", func(t *testing.T) {
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
				NewSource(TypeSQL, "00001_foo.sql", 1),
				NewSource(TypeSQL, "00002_bar.sql", 2),
				NewSource(TypeSQL, "00003_baz.sql", 3),
				NewSource(TypeSQL, "00110_qux.sql", 110),
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
				NewSource(TypeSQL, "00001_foo.sql", 1),
				NewSource(TypeSQL, "00003_baz.sql", 3),
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
		check.Equal(t, sources.sqlSources[0].Path, "1_foo.sql")
		check.Equal(t, sources.sqlSources[0].Version, int64(1))
		// 2
		check.Equal(t, sources.sqlSources[1].Path, "5_qux.sql")
		check.Equal(t, sources.sqlSources[1].Version, int64(5))
		// 3
		check.Equal(t, sources.goSources[0].Path, "4_something.go")
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
			NewSource(TypeSQL, "876_a.sql", 876),
		})
		assertDirpath("dir1", []Source{
			NewSource(TypeSQL, "101_a.sql", 101),
			NewSource(TypeSQL, "102_b.sql", 102),
			NewSource(TypeSQL, "103_c.sql", 103),
		})
		assertDirpath("dir2", []Source{
			NewSource(TypeSQL, "201_a.sql", 201),
		})
		assertDirpath("dir3", nil)
	})
}

func TestMerge(t *testing.T) {
	t.Parallel()

	t.Run("with_go_files_on_disk", func(t *testing.T) {
		mapFS := fstest.MapFS{
			// SQL
			"migrations/00001_foo.sql": sqlMapFile,
			// Go
			"migrations/00002_bar.go": {Data: []byte(`package migrations`)},
			"migrations/00003_baz.go": {Data: []byte(`package migrations`)},
		}
		fsys, err := fs.Sub(mapFS, "migrations")
		check.NoError(t, err)
		sources, err := collectFileSources(fsys, false, nil)
		check.NoError(t, err)
		check.Equal(t, len(sources.sqlSources), 1)
		check.Equal(t, len(sources.goSources), 2)
		src1 := sources.lookup(TypeSQL, 1)
		check.Bool(t, src1 != nil, true)
		src2 := sources.lookup(TypeGo, 2)
		check.Bool(t, src2 != nil, true)
		src3 := sources.lookup(TypeGo, 3)
		check.Bool(t, src3 != nil, true)

		t.Run("valid", func(t *testing.T) {
			migrations, err := merge(sources, map[int64]*goMigration{
				2: newGoMigration("", nil, nil),
				3: newGoMigration("", nil, nil),
			})
			check.NoError(t, err)
			check.Number(t, len(migrations), 3)
			assertMigration(t, migrations[0], NewSource(TypeSQL, "00001_foo.sql", 1))
			assertMigration(t, migrations[1], NewSource(TypeGo, "00002_bar.go", 2))
			assertMigration(t, migrations[2], NewSource(TypeGo, "00003_baz.go", 3))
		})
		t.Run("unregistered_all", func(t *testing.T) {
			_, err := merge(sources, nil)
			check.HasError(t, err)
			check.Contains(t, err.Error(), "error: detected 2 unregistered Go files:")
			check.Contains(t, err.Error(), "00002_bar.go")
			check.Contains(t, err.Error(), "00003_baz.go")
		})
		t.Run("unregistered_some", func(t *testing.T) {
			_, err := merge(sources, map[int64]*goMigration{
				2: newGoMigration("", nil, nil),
			})
			check.HasError(t, err)
			check.Contains(t, err.Error(), "error: detected 1 unregistered Go file")
			check.Contains(t, err.Error(), "00003_baz.go")
		})
		t.Run("duplicate_sql", func(t *testing.T) {
			_, err := merge(sources, map[int64]*goMigration{
				1: newGoMigration("", nil, nil), // duplicate. SQL already exists.
				2: newGoMigration("", nil, nil),
				3: newGoMigration("", nil, nil),
			})
			check.HasError(t, err)
			check.Contains(t, err.Error(), "found duplicate migration version 1")
		})
	})
	t.Run("no_go_files_on_disk", func(t *testing.T) {
		mapFS := fstest.MapFS{
			// SQL
			"migrations/00001_foo.sql": sqlMapFile,
			"migrations/00002_bar.sql": sqlMapFile,
			"migrations/00005_baz.sql": sqlMapFile,
		}
		fsys, err := fs.Sub(mapFS, "migrations")
		check.NoError(t, err)
		sources, err := collectFileSources(fsys, false, nil)
		check.NoError(t, err)
		t.Run("unregistered_all", func(t *testing.T) {
			migrations, err := merge(sources, map[int64]*goMigration{
				3: newGoMigration("", nil, nil),
				// 4 is missing
				6: newGoMigration("", nil, nil),
			})
			check.NoError(t, err)
			check.Number(t, len(migrations), 5)
			assertMigration(t, migrations[0], NewSource(TypeSQL, "00001_foo.sql", 1))
			assertMigration(t, migrations[1], NewSource(TypeSQL, "00002_bar.sql", 2))
			assertMigration(t, migrations[2], NewSource(TypeGo, "", 3))
			assertMigration(t, migrations[3], NewSource(TypeSQL, "00005_baz.sql", 5))
			assertMigration(t, migrations[4], NewSource(TypeGo, "", 6))
		})
	})
	t.Run("partial_go_files_on_disk", func(t *testing.T) {
		mapFS := fstest.MapFS{
			"migrations/00001_foo.sql": sqlMapFile,
			"migrations/00002_bar.go":  &fstest.MapFile{Data: []byte(`package migrations`)},
		}
		fsys, err := fs.Sub(mapFS, "migrations")
		check.NoError(t, err)
		sources, err := collectFileSources(fsys, false, nil)
		check.NoError(t, err)
		t.Run("unregistered_all", func(t *testing.T) {
			migrations, err := merge(sources, map[int64]*goMigration{
				// This is the only Go file on disk.
				2: newGoMigration("", nil, nil),
				// These are not on disk. Explicitly registered.
				3: newGoMigration("", nil, nil),
				6: newGoMigration("", nil, nil),
			})
			check.NoError(t, err)
			check.Number(t, len(migrations), 4)
			assertMigration(t, migrations[0], NewSource(TypeSQL, "00001_foo.sql", 1))
			assertMigration(t, migrations[1], NewSource(TypeGo, "00002_bar.go", 2))
			assertMigration(t, migrations[2], NewSource(TypeGo, "", 3))
			assertMigration(t, migrations[3], NewSource(TypeGo, "", 6))
		})
	})
}

func TestFindMissingMigrations(t *testing.T) {
	t.Parallel()

	t.Run("db_has_max_version", func(t *testing.T) {
		// Test case: database has migrations 1, 3, 4, 5, 7
		// Missing migrations: 2, 6
		// Filesystem has migrations 1, 2, 3, 4, 5, 6, 7, 8
		dbMigrations := []*database.ListMigrationsResult{
			{Version: 1},
			{Version: 3},
			{Version: 4},
			{Version: 5},
			{Version: 7}, // <-- database max version_id
		}
		fsMigrations := []*migration{
			newMigration(1),
			newMigration(2), // missing migration
			newMigration(3),
			newMigration(4),
			newMigration(5),
			newMigration(6), // missing migration
			newMigration(7), // ----- database max version_id -----
			newMigration(8), // new migration
		}
		got := findMissingMigrations(dbMigrations, fsMigrations)
		check.Number(t, len(got), 2)
		check.Number(t, got[0].versionID, 2)
		check.Number(t, got[1].versionID, 6)

		// Sanity check.
		check.Number(t, len(findMissingMigrations(nil, nil)), 0)
		check.Number(t, len(findMissingMigrations(dbMigrations, nil)), 0)
		check.Number(t, len(findMissingMigrations(nil, fsMigrations)), 0)
	})
	t.Run("fs_has_max_version", func(t *testing.T) {
		dbMigrations := []*database.ListMigrationsResult{
			{Version: 1},
			{Version: 5},
			{Version: 2},
		}
		fsMigrations := []*migration{
			newMigration(3), // new migration
			newMigration(4), // new migration
		}
		got := findMissingMigrations(dbMigrations, fsMigrations)
		check.Number(t, len(got), 2)
		check.Number(t, got[0].versionID, 3)
		check.Number(t, got[1].versionID, 4)
	})
}

func newMigration(version int64) *migration {
	return &migration{
		Source: Source{
			Version: version,
		},
	}
}

func assertMigration(t *testing.T, got *migration, want Source) {
	t.Helper()
	check.Equal(t, got.Source, want)
	switch got.Source.Type {
	case TypeGo:
		check.Bool(t, got.Go != nil, true)
	case TypeSQL:
		check.Bool(t, got.SQL == nil, true)
	default:
		t.Fatalf("unknown migration type: %s", got.Source.Type)
	}
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
