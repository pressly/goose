package goose

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	_ "github.com/ziutek/mymysql/godrv"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

var ErrTableDoesNotExist = errors.New("table does not exist")

type MigrationRecord struct {
	VersionId int64
	TStamp    time.Time
	IsApplied bool // was this a result of up() or down()
}

type Migration struct {
	Version  int64
	Next     int64  // next version, or -1 if none
	Previous int64  // previous version, -1 if none
	Source   string // .go or .sql script
}

type MigrationSlice []Migration

// helpers so we can use pkg sort
func (s MigrationSlice) Len() int           { return len(s) }
func (s MigrationSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s MigrationSlice) Less(i, j int) bool { return s[i].Version < s[j].Version }

type MigrationMap struct {
	Migrations MigrationSlice // migrations, sorted according to Direction
	Direction  bool           // sort direction: true -> Up, false -> Down
}

func RunMigrations(conf *DBConf, migrationsDir string, target int64) {

	db, err := sql.Open(conf.Driver.Name, conf.Driver.OpenStr)
	if err != nil {
		log.Fatal("couldn't open DB:", err)
	}
	defer db.Close()

	current, e := EnsureDBVersion(conf, db)
	if e != nil {
		log.Fatalf("couldn't get DB version: %v", e)
	}

	mm, err := CollectMigrations(migrationsDir, current, target)
	if err != nil {
		log.Fatal(err)
	}

	if len(mm.Migrations) == 0 {
		fmt.Printf("goose: no migrations to run. current version: %d\n", current)
		return
	}

	mm.Sort(current < target)

	fmt.Printf("goose: migrating db environment '%v', current version: %d, target: %d\n",
		conf.Env, current, target)

	for _, m := range mm.Migrations {

		var e error

		switch filepath.Ext(m.Source) {
		case ".go":
			e = runGoMigration(conf, m.Source, m.Version, mm.Direction)
		case ".sql":
			e = runSQLMigration(conf, db, m.Source, m.Version, mm.Direction)
		}

		if e != nil {
			log.Fatalf("FAIL %v, quitting migration", e)
		}

		fmt.Println("OK   ", filepath.Base(m.Source))
	}
}

// collect all the valid looking migration scripts in the
// migrations folder, and key them by version
func CollectMigrations(dirpath string, current, target int64) (mm *MigrationMap, err error) {

	mm = &MigrationMap{}

	// extract the numeric component of each migration,
	// filter out any uninteresting files,
	// and ensure we only have one file per migration version.
	filepath.Walk(dirpath, func(name string, info os.FileInfo, err error) error {

		if v, e := NumericComponent(name); e == nil {

			for _, m := range mm.Migrations {
				if v == m.Version {
					log.Fatalf("more than one file specifies the migration for version %d (%s and %s)",
						v, m.Source, filepath.Join(dirpath, name))
				}
			}

			if versionFilter(v, current, target) {
				mm.Append(v, name)
			}
		}

		return nil
	})

	return mm, nil
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

func (mm *MigrationMap) Append(v int64, source string) {
	mm.Migrations = append(mm.Migrations, Migration{
		Version:  v,
		Next:     -1,
		Previous: -1,
		Source:   source,
	})
}

func (mm *MigrationMap) Sort(direction bool) {
	sort.Sort(mm.Migrations)

	// set direction, and reverse order if need be
	mm.Direction = direction
	if mm.Direction == false {
		for i, j := 0, len(mm.Migrations)-1; i < j; i, j = i+1, j-1 {
			mm.Migrations[i], mm.Migrations[j] = mm.Migrations[j], mm.Migrations[i]
		}
	}

	// now that we're sorted in the appropriate direction,
	// populate next and previous for each migration
	for i, m := range mm.Migrations {
		prev := int64(-1)
		if i > 0 {
			prev = mm.Migrations[i-1].Version
			mm.Migrations[i-1].Next = m.Version
		}
		mm.Migrations[i].Previous = prev
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
func EnsureDBVersion(conf *DBConf, db *sql.DB) (int64, error) {

	rows, err := conf.Driver.Dialect.dbVersionQuery(db)
	if err != nil {
		if err == ErrTableDoesNotExist {
			return 0, createVersionTable(conf, db)
		}
		return 0, err
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

		// if version has been applied and not marked to be skipped, we're done
		if row.IsApplied && !skip {
			return row.VersionId, nil
		}

		// version is either not applied, or we've already seen a more
		// recent version of it that was not applied.
		if !skip {
			toSkip = append(toSkip, row.VersionId)
		}
	}

	panic("failure in EnsureDBVersion()")
}

// Create the goose_db_version table
// and insert the initial 0 value into it
func createVersionTable(conf *DBConf, db *sql.DB) error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}

	d := conf.Driver.Dialect

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
func GetDBVersion(conf *DBConf) (version int64, err error) {

	db, err := sql.Open(conf.Driver.Name, conf.Driver.OpenStr)
	if err != nil {
		return -1, err
	}
	defer db.Close()

	version, err = EnsureDBVersion(conf, db)
	if err != nil {
		return -1, err
	}

	return version, nil
}

func GetPreviousDBVersion(dirpath string, version int64) (previous, earliest int64) {

	previous = -1
	earliest = (1 << 63) - 1

	filepath.Walk(dirpath, func(name string, info os.FileInfo, err error) error {

		if !info.IsDir() {
			if v, e := NumericComponent(name); e == nil {
				if v > previous && v < version {
					previous = v
				}
				if v < earliest {
					earliest = v
				}
			}
		}

		return nil
	})

	return previous, earliest
}

// helper to identify the most recent possible version
// within a folder of migration scripts
func GetMostRecentDBVersion(dirpath string) (version int64, err error) {

	version = -1

	filepath.Walk(dirpath, func(name string, info os.FileInfo, walkerr error) error {

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
