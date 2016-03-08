package goose

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"
)

var (
	ErrNoPreviousVersion = errors.New("no previous version found")
	goMigrations         []*Migration
)

type MigrationRecord struct {
	VersionId int64
	TStamp    time.Time
	IsApplied bool // was this a result of up() or down()
}

type Migration struct {
	Version  int64
	Next     int64               // next version, or -1 if none
	Previous int64               // previous version, -1 if none
	Source   string              // path to .sql script
	Up       func(*sql.Tx) error // Up go migration function
	Down     func(*sql.Tx) error // Down go migration function
}

type migrationSorter []*Migration

// helpers so we can use pkg sort
func (ms migrationSorter) Len() int           { return len(ms) }
func (ms migrationSorter) Swap(i, j int)      { ms[i], ms[j] = ms[j], ms[i] }
func (ms migrationSorter) Less(i, j int) bool { return ms[i].Version < ms[j].Version }

func AddMigration(up func(*sql.Tx) error, down func(*sql.Tx) error) {
	_, filename, _, _ := runtime.Caller(1)
	v, _ := NumericComponent(filename)
	migration := &Migration{Version: v, Next: -1, Previous: -1, Up: up, Down: down, Source: filename}

	goMigrations = append(goMigrations, migration)
}

func RunMigrations(db *sql.DB, dir string, target int64) (err error) {
	current, err := EnsureDBVersion(db)
	if err != nil {
		return err
	}

	migrations, err := CollectMigrations(dir, current, target)
	if err != nil {
		return err
	}

	if len(migrations) == 0 {
		fmt.Printf("goose: no migrations to run. current version: %d\n", current)
		return nil
	}

	ms := migrationSorter(migrations)
	direction := current < target
	ms.Sort(direction)

	fmt.Printf("goose: migrating db, current version: %d, target: %d\n", current, target)

	for _, m := range ms {

		switch filepath.Ext(m.Source) {
		case ".sql":
			if err = runSQLMigration(db, m.Source, m.Version, direction); err != nil {
				return errors.New(fmt.Sprintf("FAIL %v, quitting migration", err))
			}

		case ".go":
			tx, err := db.Begin()
			if err != nil {
				log.Fatal("db.Begin: ", err)
			}

			fn := m.Up
			if !direction {
				fn = m.Down
			}
			if fn != nil {
				if err := fn(tx); err != nil {
					tx.Rollback()
					log.Fatalf("FAIL %s (%v), quitting migration.", filepath.Base(m.Source), err)
					return err
				}
			}

			if err = FinalizeMigration(tx, direction, m.Version); err != nil {
				log.Fatalf("error finalizing migration %s, quitting. (%v)", filepath.Base(m.Source), err)
			}
		}

		fmt.Println("OK   ", filepath.Base(m.Source))
	}

	return nil
}

// collect all the valid looking migration scripts in the
// migrations folder and go func registry, and key them by version
func CollectMigrations(dirpath string, current, target int64) (m []*Migration, err error) {

	// extract the numeric component of each migration,
	// filter out any uninteresting files,
	// and ensure we only have one file per migration version.
	sqlMigrations, err := filepath.Glob(dirpath + "/*.sql")
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
			m = append(m, migration)
		}
	}

	for _, migration := range goMigrations {
		v, err := NumericComponent(migration.Source)
		if err != nil {
			return nil, err
		}
		if versionFilter(v, current, target) {
			m = append(m, migration)
		}
	}

	return m, nil
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

func (ms migrationSorter) Sort(direction bool) {

	// sort ascending or descending by version
	if direction {
		sort.Sort(ms)
	} else {
		sort.Sort(sort.Reverse(ms))
	}

	// now that we're sorted in the appropriate direction,
	// populate next and previous for each migration
	for i, m := range ms {
		prev := int64(-1)
		if i > 0 {
			prev = ms[i-1].Version
			ms[i-1].Next = m.Version
		}
		ms[i].Previous = prev
	}
}

// look for migration scripts with names in the form:
//  XXX_descriptivename.ext
// where XXX specifies the version number
// and ext specifies the type of migration
func NumericComponent(name string) (int64, error) {

	base := filepath.Base(name)

	if ext := filepath.Ext(base); ext != ".go" && ext != ".sql" {
		return 0, errors.New("not a recognized migration file type")
	}

	idx := strings.Index(base, "_")
	if idx < 0 {
		return 0, errors.New("no separator found")
	}

	n, e := strconv.ParseInt(base[:idx], 10, 64)
	if e == nil && n <= 0 {
		return 0, errors.New("migration IDs must be greater than zero")
	}

	return n, e
}

// retrieve the current version for this DB.
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
		if err = rows.Scan(&row.VersionId, &row.IsApplied); err != nil {
			log.Fatal("error scanning rows:", err)
		}

		// have we already marked this version to be skipped?
		skip := false
		for _, v := range toSkip {
			if v == row.VersionId {
				skip = true
				break
			}
		}

		if skip {
			continue
		}

		// if version has been applied we're done
		if row.IsApplied {
			return row.VersionId, nil
		}

		// latest version of migration has not been applied.
		toSkip = append(toSkip, row.VersionId)
	}

	panic("failure in EnsureDBVersion()")
}

// Create the goose_db_version table
// and insert the initial 0 value into it
func createVersionTable(db *sql.DB) error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}

	d := GetDialect()

	if _, err := txn.Exec(d.createVersionTableSql()); err != nil {
		txn.Rollback()
		return err
	}

	version := 0
	applied := true
	if _, err := txn.Exec(d.insertVersionSql(), version, applied); err != nil {
		txn.Rollback()
		return err
	}

	return txn.Commit()
}

// wrapper for EnsureDBVersion for callers that don't already have
// their own DB instance
func GetDBVersion(db *sql.DB) (int64, error) {
	version, err := EnsureDBVersion(db)
	if err != nil {
		return -1, err
	}

	return version, nil
}

func GetPreviousDBVersion(dirpath string, version int64) (previous int64, err error) {

	previous = -1
	sawGivenVersion := false

	filepath.Walk(dirpath, func(name string, info os.FileInfo, walkerr error) error {

		if !info.IsDir() {
			if v, e := NumericComponent(name); e == nil {
				if v > previous && v < version {
					previous = v
				}
				if v == version {
					sawGivenVersion = true
				}
			}
		}

		return nil
	})

	if previous == -1 {
		if sawGivenVersion {
			// the given version is (likely) valid but we didn't find
			// anything before it.
			// 'previous' must reflect that no migrations have been applied.
			previous = 0
		} else {
			err = ErrNoPreviousVersion
		}
	}

	return
}

// helper to identify the most recent possible version
// within a folder of migration scripts
func GetMostRecentDBVersion(dirpath string) (version int64, err error) {

	version = -1

	filepath.Walk(dirpath, func(name string, info os.FileInfo, walkerr error) error {
		if walkerr != nil {
			return walkerr
		}

		if !info.IsDir() {
			if v, e := NumericComponent(name); e == nil {
				if v > version {
					version = v
				}
			}
		}

		return nil
	})

	if version == -1 {
		err = errors.New("no valid version found")
	}

	return
}

func CreateMigration(name, migrationType, dir string, t time.Time) (path string, err error) {

	if migrationType != "go" && migrationType != "sql" {
		return "", errors.New("migration type must be 'go' or 'sql'")
	}

	timestamp := t.Format("20060102150405")
	filename := fmt.Sprintf("%v_%v.%v", timestamp, name, migrationType)

	fpath := filepath.Join(dir, filename)
	tmpl := sqlMigrationTemplate

	path, err = writeTemplateToFile(fpath, tmpl, timestamp)

	return
}

// Update the version table for the given migration,
// and finalize the transaction.
func FinalizeMigration(tx *sql.Tx, direction bool, v int64) error {

	// XXX: drop goose_db_version table on some minimum version number?
	stmt := GetDialect().insertVersionSql()
	if _, err := tx.Exec(stmt, v, direction); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

var sqlMigrationTemplate = template.Must(template.New("goose.sql-migration").Parse(`
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

`))
