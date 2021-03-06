// Package iofs contains functions that allows to work with migration using io/fs package.
package iofs

import (
	"fmt"
	"io/fs"
	"os"
	"sort"

	"github.com/pressly/goose"
)

// CollectMigrations returns all the valid looking migration scripts in the
// migrations folder and key them by version.
func CollectMigrations(fsys fs.FS, dirpath string, current, target int64) (goose.Migrations, error) {
	if _, err := fs.Stat(fsys, dirpath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%s directory does not exist", dirpath)
	}

	var migrations goose.Migrations

	// SQL migration files.
	sqlMigrationFiles, err := fs.Glob(fsys, dirpath + "/*.sql")
	if err != nil {
		return nil, err
	}
	for _, file := range sqlMigrationFiles {
		v, err := goose.NumericComponent(file)
		if err != nil {
			return nil, err
		}
		if versionFilter(v, current, target) {
			f, err := fsys.Open(file)
			if err != nil {
				return nil, err
			}

			migration := &goose.Migration{Version: v, Next: -1, Previous: -1, Source: file, SourceReader: f}
			migrations = append(migrations, migration)
		}
	}

	migrations = sortAndConnectMigrations(migrations)

	return migrations, nil
}

func sortAndConnectMigrations(migrations goose.Migrations) goose.Migrations {
	sort.Sort(migrations)

	// now that we're sorted in the appropriate direction,
	// populate next and previous for each migration
	for i, m := range migrations {
		prev := int64(-1)
		if i > 0 {
			prev = migrations[i-1].Version
			migrations[i-1].Next = m.Version
		}
		migrations[i].Previous = prev
	}

	return migrations
}

func versionFilter(v, current, target int64) bool {
	if target > current {
		return v > current && v <= target
	}

	if target < current {
		return v <= current && v > target
	}

	return false
}
