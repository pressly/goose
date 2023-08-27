package goose

import (
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3/internal/check"
)

func TestMigrationSort(t *testing.T) {
	t.Parallel()

	ms := Migrations{}

	// insert in any order
	ms = append(ms, newMigration(20120000, "test"))
	ms = append(ms, newMigration(20128000, "test"))
	ms = append(ms, newMigration(20129000, "test"))
	ms = append(ms, newMigration(20127000, "test"))

	ms = sortAndConnectMigrations(ms)

	sorted := []int64{20120000, 20127000, 20128000, 20129000}

	validateMigrationSort(t, ms, sorted)
}

func newMigration(v int64, src string) *Migration {
	return &Migration{Version: v, Previous: -1, Next: -1, Source: src}
}

func validateMigrationSort(t *testing.T, ms Migrations, sorted []int64) {
	for i, m := range ms {
		if sorted[i] != m.Version {
			t.Error("incorrect sorted version")
		}

		var next, prev int64

		if i == 0 {
			prev = -1
			next = ms[i+1].Version
		} else if i == len(ms)-1 {
			prev = ms[i-1].Version
			next = -1
		} else {
			prev = ms[i-1].Version
			next = ms[i+1].Version
		}

		if m.Next != next {
			t.Errorf("mismatched Next. v: %v, got %v, wanted %v\n", m, m.Next, next)
		}

		if m.Previous != prev {
			t.Errorf("mismatched Previous v: %v, got %v, wanted %v\n", m, m.Previous, prev)
		}
	}

	t.Log(ms)
}

func TestCollectMigrations(t *testing.T) {
	// Not safe to run in parallel
	t.Run("no_migration_files_found", func(t *testing.T) {
		tmp := t.TempDir()
		err := os.MkdirAll(filepath.Join(tmp, "migrations-test"), 0755)
		check.NoError(t, err)
		_, err = collectMigrationsFS(os.DirFS(tmp), "migrations-test", 0, math.MaxInt64, nil)
		check.HasError(t, err)
		check.Contains(t, err.Error(), "no migration files found")
	})
	t.Run("filesystem_registered_with_single_dirpath", func(t *testing.T) {
		t.Cleanup(func() { clearMap(registeredGoMigrations) })
		file1, file2 := "09081_a.go", "09082_b.go"
		file3, file4 := "19081_a.go", "19082_b.go"
		AddNamedMigrationContext(file1, nil, nil)
		AddNamedMigrationContext(file2, nil, nil)
		check.Number(t, len(registeredGoMigrations), 2)
		tmp := t.TempDir()
		dir := filepath.Join(tmp, "migrations", "dir1")
		err := os.MkdirAll(dir, 0755)
		check.NoError(t, err)
		createEmptyFile(t, dir, file1)
		createEmptyFile(t, dir, file2)
		createEmptyFile(t, dir, file3)
		createEmptyFile(t, dir, file4)
		fsys := os.DirFS(tmp)
		files, err := fs.ReadDir(fsys, "migrations/dir1")
		check.NoError(t, err)
		check.Number(t, len(files), 4)
		all, err := collectMigrationsFS(fsys, "migrations/dir1", 0, math.MaxInt64, registeredGoMigrations)
		check.NoError(t, err)
		check.Number(t, len(all), 4)
		check.Number(t, all[0].Version, 9081)
		check.Number(t, all[1].Version, 9082)
		check.Number(t, all[2].Version, 19081)
		check.Number(t, all[3].Version, 19082)
	})
	t.Run("filesystem_registered_with_multiple_dirpath", func(t *testing.T) {
		t.Cleanup(func() { clearMap(registeredGoMigrations) })
		file1, file2, file3 := "00001_a.go", "00002_b.go", "01111_c.go"
		AddNamedMigrationContext(file1, nil, nil)
		AddNamedMigrationContext(file2, nil, nil)
		AddNamedMigrationContext(file3, nil, nil)
		check.Number(t, len(registeredGoMigrations), 3)
		tmp := t.TempDir()
		dir1 := filepath.Join(tmp, "migrations", "dir1")
		dir2 := filepath.Join(tmp, "migrations", "dir2")
		err := os.MkdirAll(dir1, 0755)
		check.NoError(t, err)
		err = os.MkdirAll(dir2, 0755)
		check.NoError(t, err)
		createEmptyFile(t, dir1, file1)
		createEmptyFile(t, dir1, file2)
		createEmptyFile(t, dir2, file3)
		fsys := os.DirFS(tmp)
		// Validate if dirpath 1 is specified we get the two Go migrations in migrations/dir1 folder
		// even though 3 Go migrations have been registered.
		{
			all, err := collectMigrationsFS(fsys, "migrations/dir1", 0, math.MaxInt64, registeredGoMigrations)
			check.NoError(t, err)
			check.Number(t, len(all), 2)
			check.Number(t, all[0].Version, 1)
			check.Number(t, all[1].Version, 2)
		}
		// Validate if dirpath 2 is specified we only get the one Go migration in migrations/dir2 folder
		// even though 3 Go migrations have been registered.
		{
			all, err := collectMigrationsFS(fsys, "migrations/dir2", 0, math.MaxInt64, registeredGoMigrations)
			check.NoError(t, err)
			check.Number(t, len(all), 1)
			check.Number(t, all[0].Version, 1111)
		}
	})
	t.Run("empty_filesystem_registered_manually", func(t *testing.T) {
		t.Cleanup(func() { clearMap(registeredGoMigrations) })
		AddNamedMigrationContext("00101_a.go", nil, nil)
		AddNamedMigrationContext("00102_b.go", nil, nil)
		check.Number(t, len(registeredGoMigrations), 2)
		tmp := t.TempDir()
		err := os.MkdirAll(filepath.Join(tmp, "migrations"), 0755)
		check.NoError(t, err)
		all, err := collectMigrationsFS(os.DirFS(tmp), "migrations", 0, math.MaxInt64, registeredGoMigrations)
		check.NoError(t, err)
		check.Number(t, len(all), 2)
		check.Number(t, all[0].Version, 101)
		check.Number(t, all[1].Version, 102)
	})
	t.Run("unregistered_go_migrations", func(t *testing.T) {
		t.Cleanup(func() { clearMap(registeredGoMigrations) })
		file1, file2, file3 := "00001_a.go", "00998_b.go", "00999_c.go"
		// Only register file1 and file3, somehow user forgot to init in the
		// valid looking file2 Go migration
		AddNamedMigrationContext(file1, nil, nil)
		AddNamedMigrationContext(file3, nil, nil)
		check.Number(t, len(registeredGoMigrations), 2)
		tmp := t.TempDir()
		dir1 := filepath.Join(tmp, "migrations", "dir1")
		err := os.MkdirAll(dir1, 0755)
		check.NoError(t, err)
		// Include the valid file2 with file1, file3. But remember, it has NOT been
		// registered.
		createEmptyFile(t, dir1, file1)
		createEmptyFile(t, dir1, file2)
		createEmptyFile(t, dir1, file3)
		all, err := collectMigrationsFS(os.DirFS(tmp), "migrations/dir1", 0, math.MaxInt64, registeredGoMigrations)
		check.NoError(t, err)
		check.Number(t, len(all), 3)
		check.Number(t, all[0].Version, 1)
		check.Bool(t, all[0].Registered, true)
		check.Number(t, all[1].Version, 998)
		// This migrations is marked unregistered and will lazily raise an error if/when this
		// migration is run
		check.Bool(t, all[1].Registered, false)
		check.Number(t, all[2].Version, 999)
		check.Bool(t, all[2].Registered, true)
	})
	t.Run("with_skipped_go_files", func(t *testing.T) {
		t.Cleanup(func() { clearMap(registeredGoMigrations) })
		file1, file2, file3, file4 := "00001_a.go", "00002_b.sql", "00999_c_test.go", "embed.go"
		AddNamedMigrationContext(file1, nil, nil)
		check.Number(t, len(registeredGoMigrations), 1)
		tmp := t.TempDir()
		dir1 := filepath.Join(tmp, "migrations", "dir1")
		err := os.MkdirAll(dir1, 0755)
		check.NoError(t, err)
		createEmptyFile(t, dir1, file1)
		createEmptyFile(t, dir1, file2)
		createEmptyFile(t, dir1, file3)
		createEmptyFile(t, dir1, file4)
		all, err := collectMigrationsFS(os.DirFS(tmp), "migrations/dir1", 0, math.MaxInt64, registeredGoMigrations)
		check.NoError(t, err)
		check.Number(t, len(all), 2)
		check.Number(t, all[0].Version, 1)
		check.Bool(t, all[0].Registered, true)
		check.Number(t, all[1].Version, 2)
		check.Bool(t, all[1].Registered, false)
	})
	t.Run("current_and_target", func(t *testing.T) {
		t.Cleanup(func() { clearMap(registeredGoMigrations) })
		file1, file2, file3 := "01001_a.go", "01002_b.sql", "01003_c.go"
		AddNamedMigrationContext(file1, nil, nil)
		AddNamedMigrationContext(file3, nil, nil)
		check.Number(t, len(registeredGoMigrations), 2)
		tmp := t.TempDir()
		dir1 := filepath.Join(tmp, "migrations", "dir1")
		err := os.MkdirAll(dir1, 0755)
		check.NoError(t, err)
		createEmptyFile(t, dir1, file1)
		createEmptyFile(t, dir1, file2)
		createEmptyFile(t, dir1, file3)
		all, err := collectMigrationsFS(os.DirFS(tmp), "migrations/dir1", 1001, 1003, registeredGoMigrations)
		check.NoError(t, err)
		check.Number(t, len(all), 2)
		check.Number(t, all[0].Version, 1002)
		check.Number(t, all[1].Version, 1003)
	})
}

func TestVersionFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		v       int64
		current int64
		target  int64
		want    bool
	}{
		{2, 1, 3, true},  // v is within the range
		{4, 1, 3, false}, // v is outside the range
		{2, 3, 1, true},  // v is within the reversed range
		{4, 3, 1, false}, // v is outside the reversed range
		{3, 1, 3, true},  // v is equal to target
		{1, 1, 3, false}, // v is equal to current, not within the range
		{1, 3, 1, false}, // v is equal to current, not within the reversed range
		// Always return false if current equal target
		{1, 2, 2, false},
		{2, 2, 2, false},
		{3, 2, 2, false},
	}
	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			got := versionFilter(tc.v, tc.current, tc.target)
			if got != tc.want {
				t.Errorf("versionFilter(%d, %d, %d) = %v, want %v", tc.v, tc.current, tc.target, got, tc.want)
			}
		})
	}
}
func createEmptyFile(t *testing.T, dir, name string) {
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	check.NoError(t, err)
	defer f.Close()
}

func clearMap(m map[int64]*Migration) {
	for k := range m {
		delete(m, k)
	}
}
