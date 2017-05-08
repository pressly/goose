package goose

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sort"
)

var (
	// ErrNoCurrentVersion when a current migration version is not found.
	ErrNoCurrentVersion = errors.New("no current version found")
	// ErrNoNextVersion when the next migration version is not found.
	ErrNoNextVersion = errors.New("no next version found")
	// MaxVersion is the maximum allowed version.
	MaxVersion int64 = 9223372036854775807 // max(int64)

	goMigrations []*Migration
)

// Migrations slice.
type Migrations []*Migration

// helpers so we can use pkg sort
func (ms Migrations) Len() int      { return len(ms) }
func (ms Migrations) Swap(i, j int) { ms[i], ms[j] = ms[j], ms[i] }
func (ms Migrations) Less(i, j int) bool {
	if ms[i].Version == ms[j].Version {
		log.Fatalf("goose: duplicate version %v detected:\n%v\n%v", ms[i].Version, ms[i].Source, ms[j].Source)
	}
	return ms[i].Version < ms[j].Version
}

// Current gets the current migration.
func (ms Migrations) Current(current int64) (*Migration, error) {
	for i, migration := range ms {
		if migration.Version == current {
			return ms[i], nil
		}
	}

	return nil, ErrNoCurrentVersion
}

// Next gets the next migration.
func (ms Migrations) Next(current int64) (*Migration, error) {
	for i, migration := range ms {
		if migration.Version > current {
			return ms[i], nil
		}
	}

	return nil, ErrNoNextVersion
}

// Previous : Get the previous migration.
func (ms Migrations) Previous(current int64) (*Migration, error) {
	for i := len(ms) - 1; i >= 0; i-- {
		if ms[i].Version < current {
			return ms[i], nil
		}
	}

	return nil, ErrNoNextVersion
}

// Last gets the last migration.
func (ms Migrations) Last() (*Migration, error) {
	if len(ms) == 0 {
		return nil, ErrNoNextVersion
	}

	return ms[len(ms)-1], nil
}

func (ms Migrations) String() string {
	str := ""
	for _, m := range ms {
		str += fmt.Sprintln(m)
	}
	return str
}

// AddMigration adds a migration.
func AddMigration(up func(*sql.Tx) error, down func(*sql.Tx) error) {
	_, filename, _, _ := runtime.Caller(1)
	AddNamedMigration(filename, up, down)
}

// AddNamedMigration : Add a named migration.
func AddNamedMigration(filename string, up func(*sql.Tx) error, down func(*sql.Tx) error) {
	v, _ := NumericComponent(filename)
	migration := &Migration{Version: v, Next: -1, Previous: -1, UpFn: up, DownFn: down, Source: filename}

	goMigrations = append(goMigrations, migration)
}

// CollectMigrations returns all the valid looking migration scripts in the
// migrations folder and go func registry, and key them by version.
func CollectMigrations(dirpath string, current, target int64) (Migrations, error) {
	var migrations Migrations

	// extract the numeric component of each migration,
	// filter out any uninteresting files,
	// and ensure we only have one file per migration version.
	sqlMigrations, err := filepath.Glob(dirpath + "/*.sql")
	sqlMigrationsSubDirectories, err := filepath.Glob(dirpath + "/**/*.sql")
	sqlMigrations = append(sqlMigrations, sqlMigrationsSubDirectories...)

	if err != nil {
		return nil, err
	}

	for _, file := range sqlMigrations {
		v, err := NumericComponent(file)
		if err != nil {
			return nil, err
		}
		if versionFilter(v, current, target) {
			migration := &Migration{Version: v, Next: -1, Previous: -1, Source: file}
			migrations = append(migrations, migration)
		}
	}

	for _, migration := range goMigrations {
		v, err := NumericComponent(migration.Source)
		if err != nil {
			return nil, err
		}
		if versionFilter(v, current, target) {
			migrations = append(migrations, migration)
		}
	}

	migrations = sortAndConnectMigrations(migrations)

	return migrations, nil
}

func sortAndConnectMigrations(migrations Migrations) Migrations {
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

// EnsureDBVersion retrieves the current version for this DB.
// Create and initialize the DB version table if it doesn't exist.
func EnsureDBVersion(db *sql.DB) (int64, error) {
	rows, err := GetDialect().dbVersionQuery(db)
	if err != nil {
		return 0, createVersionTable(db)
	}
	defer rows.Close()

	// The most recent record for each migration specifies
	// whether it has been applied or rolled back.
	// The first version we find that has been applied is the current version.

	toSkip := make([]int64, 0)

	for rows.Next() {
		var row MigrationRecord
		if err = rows.Scan(&row.VersionID, &row.IsApplied); err != nil {
			log.Fatal("error scanning rows:", err)
		}

		// have we already marked this version to be skipped?
		skip := false
		for _, v := range toSkip {
			if v == row.VersionID {
				skip = true
				break
			}
		}

		if skip {
			continue
		}

		// if version has been applied we're done
		if row.IsApplied {
			return row.VersionID, nil
		}

		// latest version of migration has not been applied.
		toSkip = append(toSkip, row.VersionID)
	}

	return 0, ErrNoNextVersion
}

// Create the goose_db_version table
// and insert the initial 0 value into it
func createVersionTable(db *sql.DB) error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}

	d := GetDialect()

	if _, err := txn.Exec(d.createVersionTableSQL()); err != nil {
		txn.Rollback()
		return err
	}

	version := 0
	applied := true
	if _, err := txn.Exec(d.insertVersionSQL(), version, applied); err != nil {
		txn.Rollback()
		return err
	}

	return txn.Commit()
}

// GetDBVersion is a wrapper for EnsureDBVersion for callers that don't already
// have their own DB instance
func GetDBVersion(db *sql.DB) (int64, error) {
	version, err := EnsureDBVersion(db)
	if err != nil {
		return -1, err
	}

	return version, nil
}
