package goose

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestCollectFileSources(t *testing.T) {
	t.Parallel()
	t.Run("nil_fsys", func(t *testing.T) {
		sources, err := collectFilesystemSources(nil, false, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, sources)
		require.Empty(t, sources.goSources)
		require.Empty(t, sources.sqlSources)
	})
	t.Run("noop_fsys", func(t *testing.T) {
		sources, err := collectFilesystemSources(noopFS{}, false, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, sources)
		require.Empty(t, sources.goSources)
		require.Empty(t, sources.sqlSources)
	})
	t.Run("empty_fsys", func(t *testing.T) {
		sources, err := collectFilesystemSources(fstest.MapFS{}, false, nil, nil)
		require.NoError(t, err)
		require.Empty(t, sources.goSources)
		require.Empty(t, sources.sqlSources)
		require.NotNil(t, sources)
	})
	t.Run("incorrect_fsys", func(t *testing.T) {
		mapFS := fstest.MapFS{
			"00000_foo.sql": sqlMapFile,
		}
		// strict disable - should not error
		sources, err := collectFilesystemSources(mapFS, false, nil, nil)
		require.NoError(t, err)
		require.Empty(t, sources.goSources)
		require.Empty(t, sources.sqlSources)
		// strict enabled - should error
		_, err = collectFilesystemSources(mapFS, true, nil, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "migration version must be greater than zero")
	})
	t.Run("collect", func(t *testing.T) {
		fsys, err := fs.Sub(newSQLOnlyFS(), "migrations")
		require.NoError(t, err)
		sources, err := collectFilesystemSources(fsys, false, nil, nil)
		require.NoError(t, err)
		require.Len(t, sources.sqlSources, 4)
		require.Empty(t, sources.goSources)
		expected := fileSources{
			sqlSources: []Source{
				newSource(TypeSQL, "00001_foo.sql", 1),
				newSource(TypeSQL, "00002_bar.sql", 2),
				newSource(TypeSQL, "00003_baz.sql", 3),
				newSource(TypeSQL, "00110_qux.sql", 110),
			},
		}
		for i := 0; i < len(sources.sqlSources); i++ {
			require.Equal(t, sources.sqlSources[i], expected.sqlSources[i])
		}
	})
	t.Run("excludes", func(t *testing.T) {
		fsys, err := fs.Sub(newSQLOnlyFS(), "migrations")
		require.NoError(t, err)
		sources, err := collectFilesystemSources(
			fsys,
			false,
			// exclude 2 files explicitly
			map[string]bool{
				"00002_bar.sql": true,
				"00110_qux.sql": true,
			},
			nil,
		)
		require.NoError(t, err)
		require.Len(t, sources.sqlSources, 2)
		require.Empty(t, sources.goSources)
		expected := fileSources{
			sqlSources: []Source{
				newSource(TypeSQL, "00001_foo.sql", 1),
				newSource(TypeSQL, "00003_baz.sql", 3),
			},
		}
		for i := 0; i < len(sources.sqlSources); i++ {
			require.Equal(t, sources.sqlSources[i], expected.sqlSources[i])
		}
	})
	t.Run("strict", func(t *testing.T) {
		mapFS := newSQLOnlyFS()
		// Add a file with no version number
		mapFS["migrations/not_valid.sql"] = &fstest.MapFile{Data: []byte("invalid")}
		fsys, err := fs.Sub(mapFS, "migrations")
		require.NoError(t, err)
		_, err = collectFilesystemSources(fsys, true, nil, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), `failed to parse numeric component from "not_valid.sql"`)
	})
	t.Run("skip_go_test_files", func(t *testing.T) {
		mapFS := fstest.MapFS{
			"1_foo.sql":     sqlMapFile,
			"2_bar.sql":     sqlMapFile,
			"3_baz.sql":     sqlMapFile,
			"4_qux.sql":     sqlMapFile,
			"5_foo_test.go": {Data: []byte(`package goose_test`)},
		}
		sources, err := collectFilesystemSources(mapFS, false, nil, nil)
		require.NoError(t, err)
		require.Len(t, sources.sqlSources, 4)
		require.Empty(t, sources.goSources)
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
		sources, err := collectFilesystemSources(mapFS, false, nil, nil)
		require.NoError(t, err)
		require.Len(t, sources.sqlSources, 2)
		require.Len(t, sources.goSources, 1)
		// 1
		require.Equal(t, "1_foo.sql", sources.sqlSources[0].Path)
		require.EqualValues(t, 1, sources.sqlSources[0].Version)
		// 2
		require.Equal(t, "5_qux.sql", sources.sqlSources[1].Path)
		require.EqualValues(t, 5, sources.sqlSources[1].Version)
		// 3
		require.Equal(t, "4_something.go", sources.goSources[0].Path)
		require.EqualValues(t, 4, sources.goSources[0].Version)
	})
	t.Run("duplicate_versions", func(t *testing.T) {
		mapFS := fstest.MapFS{
			"001_foo.sql": sqlMapFile,
			"01_bar.sql":  sqlMapFile,
		}
		_, err := collectFilesystemSources(mapFS, false, nil, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "found duplicate migration version 1")
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
			require.NoError(t, err)
			got, err := collectFilesystemSources(f, false, nil, nil)
			require.NoError(t, err)
			require.Equal(t, len(got.sqlSources), len(sqlSources))
			require.Empty(t, got.goSources)
			for i := 0; i < len(got.sqlSources); i++ {
				require.Equal(t, got.sqlSources[i], sqlSources[i])
			}
		}
		assertDirpath(".", []Source{
			newSource(TypeSQL, "876_a.sql", 876),
		})
		assertDirpath("dir1", []Source{
			newSource(TypeSQL, "101_a.sql", 101),
			newSource(TypeSQL, "102_b.sql", 102),
			newSource(TypeSQL, "103_c.sql", 103),
		})
		assertDirpath("dir2", []Source{
			newSource(TypeSQL, "201_a.sql", 201),
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
		require.NoError(t, err)
		sources, err := collectFilesystemSources(fsys, false, nil, nil)
		require.NoError(t, err)
		require.Len(t, sources.sqlSources, 1)
		require.Len(t, sources.goSources, 2)
		t.Run("valid", func(t *testing.T) {
			registered := map[int64]*Migration{
				2: NewGoMigration(2, nil, nil),
				3: NewGoMigration(3, nil, nil),
			}
			migrations, err := merge(sources, registered)
			require.NoError(t, err)
			require.Len(t, migrations, 3)
			assertMigration(t, migrations[0], newSource(TypeSQL, "00001_foo.sql", 1))
			assertMigration(t, migrations[1], newSource(TypeGo, "00002_bar.go", 2))
			assertMigration(t, migrations[2], newSource(TypeGo, "00003_baz.go", 3))
		})
		t.Run("unregistered_all", func(t *testing.T) {
			_, err := merge(sources, nil)
			require.Error(t, err)
			require.Contains(t, err.Error(), "error: detected 2 unregistered Go files:")
			require.Contains(t, err.Error(), "00002_bar.go")
			require.Contains(t, err.Error(), "00003_baz.go")
		})
		t.Run("unregistered_some", func(t *testing.T) {
			_, err := merge(sources, map[int64]*Migration{2: NewGoMigration(2, nil, nil)})
			require.Error(t, err)
			require.Contains(t, err.Error(), "error: detected 1 unregistered Go file")
			require.Contains(t, err.Error(), "00003_baz.go")
		})
		t.Run("duplicate_sql", func(t *testing.T) {
			_, err := merge(sources, map[int64]*Migration{
				1: NewGoMigration(1, nil, nil), // duplicate. SQL already exists.
				2: NewGoMigration(2, nil, nil),
				3: NewGoMigration(3, nil, nil),
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), "found duplicate migration version 1")
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
		require.NoError(t, err)
		sources, err := collectFilesystemSources(fsys, false, nil, nil)
		require.NoError(t, err)
		t.Run("unregistered_all", func(t *testing.T) {
			migrations, err := merge(sources, map[int64]*Migration{
				3: NewGoMigration(3, nil, nil),
				// 4 is missing
				6: NewGoMigration(6, nil, nil),
			})
			require.NoError(t, err)
			require.Len(t, migrations, 5)
			assertMigration(t, migrations[0], newSource(TypeSQL, "00001_foo.sql", 1))
			assertMigration(t, migrations[1], newSource(TypeSQL, "00002_bar.sql", 2))
			assertMigration(t, migrations[2], newSource(TypeGo, "", 3))
			assertMigration(t, migrations[3], newSource(TypeSQL, "00005_baz.sql", 5))
			assertMigration(t, migrations[4], newSource(TypeGo, "", 6))
		})
	})
	t.Run("partial_go_files_on_disk", func(t *testing.T) {
		mapFS := fstest.MapFS{
			"migrations/00001_foo.sql": sqlMapFile,
			"migrations/00002_bar.go":  &fstest.MapFile{Data: []byte(`package migrations`)},
		}
		fsys, err := fs.Sub(mapFS, "migrations")
		require.NoError(t, err)
		sources, err := collectFilesystemSources(fsys, false, nil, nil)
		require.NoError(t, err)
		t.Run("unregistered_all", func(t *testing.T) {
			migrations, err := merge(sources, map[int64]*Migration{
				// This is the only Go file on disk.
				2: NewGoMigration(2, nil, nil),
				// These are not on disk. Explicitly registered.
				3: NewGoMigration(3, nil, nil),
				6: NewGoMigration(6, nil, nil),
			})
			require.NoError(t, err)
			require.Len(t, migrations, 4)
			assertMigration(t, migrations[0], newSource(TypeSQL, "00001_foo.sql", 1))
			assertMigration(t, migrations[1], newSource(TypeGo, "00002_bar.go", 2))
			assertMigration(t, migrations[2], newSource(TypeGo, "", 3))
			assertMigration(t, migrations[3], newSource(TypeGo, "", 6))
		})
	})
}

func assertMigration(t *testing.T, got *Migration, want Source) {
	t.Helper()
	require.Equal(t, want.Type, got.Type)
	require.Equal(t, want.Version, got.Version)
	require.Equal(t, want.Path, got.Source)
	switch got.Type {
	case TypeGo:
		require.NotNil(t, got.goUp)
		require.NotNil(t, got.goDown)
	case TypeSQL:
		require.False(t, got.sql.Parsed)
	default:
		t.Fatalf("unknown migration type: %s", got.Type)
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

func newSource(t MigrationType, fullpath string, version int64) Source {
	return Source{
		Type:    t,
		Path:    fullpath,
		Version: version,
	}
}

var (
	sqlMapFile = &fstest.MapFile{Data: []byte(`-- +goose Up`)}
)
