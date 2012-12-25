package main

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/bmizerany/pq"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type DBVersion struct {
	VersionId int
	TStamp    time.Time
}

type Migration struct {
	Next     int    // next version, or -1 if none
	Previous int    // previous version, -1 if none
	Source   string // .go or .sql script
}

type MigrationMap struct {
	Versions   []int             // sorted slice of version keys
	Migrations map[int]Migration // sources (.sql or .go) keyed by version
	Direction  bool              // sort direction: true -> Up, false -> Down
}

func runMigrations(conf *DBConf, migrationsDir string, target int) {

	db, err := sql.Open(conf.Driver, conf.OpenStr)
	if err != nil {
		log.Fatal("couldn't open DB:", err)
	}
	defer db.Close()

	current, e := ensureDBVersion(db)
	if e != nil {
		log.Fatalf("couldn't get DB version: %v", e)
	}

	mm, err := collectMigrations(migrationsDir, current, target)
	if err != nil {
		log.Fatal(err)
	}

	if len(mm.Versions) == 0 {
		fmt.Printf("goose: no migrations to run. current version: %d\n", current)
		return
	}

	mm.Sort(current < target)

	fmt.Printf("goose: migrating db environment '%v', current version: %d, target: %d\n",
		conf.Env, current, target)

	for _, v := range mm.Versions {

		var numStatements int
		var e error

		filepath := mm.Migrations[v].Source

		switch path.Ext(filepath) {
		case ".go":
			numStatements, e = runGoMigration(conf, filepath, v, mm.Direction)
		case ".sql":
			numStatements, e = runSQLMigration(db, filepath, v, mm.Direction)
		}

		if e != nil {
			log.Fatalf("FAIL %v, quitting migration", e)
		}

		fmt.Printf("OK   %s (%d statements)\n", path.Base(filepath), numStatements)
	}
}

// collect all the valid looking migration scripts in the 
// migrations folder, and key them by version
func collectMigrations(dirpath string, current, target int) (mm *MigrationMap, err error) {

	mm = &MigrationMap{
		Migrations: make(map[int]Migration),
	}

	// extract the numeric component of each migration,
	// filter out any uninteresting files,
	// and ensure we only have one file per migration version.
	filepath.Walk(dirpath, func(name string, info os.FileInfo, err error) error {

		if v, e := numericComponent(name); e == nil {

			if _, ok := mm.Migrations[v]; ok {
				log.Fatalf("more than one file specifies the migration for version %d (%s and %s)",
					v, mm.Versions[v], path.Join(dirpath, name))
			}

			if versionFilter(v, current, target) {
				mm.Append(v, name)
			}
		}

		return nil
	})

	return mm, nil
}

func versionFilter(v, current, target int) bool {

	// special case - default target value
	if target < 0 {
		return v > current
	}

	if target > current {
		return v > current && v <= target
	}

	if target < current {
		return v <= current && v > target
	}

	return false
}

func (m *MigrationMap) Append(v int, source string) {
	m.Versions = append(m.Versions, v)
	m.Migrations[v] = Migration{
		Next:     -1,
		Previous: -1,
		Source:   source,
	}
}

func (m *MigrationMap) Sort(direction bool) {
	sort.Ints(m.Versions)

	// set direction, and reverse order if need be
	m.Direction = direction
	if m.Direction == false {
		for i, j := 0, len(m.Versions)-1; i < j; i, j = i+1, j-1 {
			m.Versions[i], m.Versions[j] = m.Versions[j], m.Versions[i]
		}
	}

	// now that we're sorted in the appropriate direction,
	// populate next and previous for each migration
	//
	// work around http://code.google.com/p/go/issues/detail?id=3117
	previousV := -1
	for _, v := range m.Versions {
		cur := m.Migrations[v]
		cur.Previous = previousV
		m.Migrations[v] = cur

		// if a migration exists at prev, its next is now v
		if prev, ok := m.Migrations[previousV]; ok {
			prev.Next = v
			m.Migrations[previousV] = prev
		}

		previousV = v
	}
}

// look for migration scripts with names in the form:
//  XXX_descriptivename.ext
// where XXX specifies the version number
// and ext specifies the type of migration
func numericComponent(name string) (int, error) {

	base := path.Base(name)

	if ext := path.Ext(base); ext != ".go" && ext != ".sql" {
		return 0, errors.New("not a recognized migration file type")
	}

	idx := strings.Index(base, "_")
	if idx < 0 {
		return 0, errors.New("no separator found")
	}

	n, e := strconv.Atoi(base[:idx])
	if e == nil && n == 0 {
		return 0, errors.New("0 is not a valid migration ID")
	}

	return n, e
}

// retrieve the current version for this DB.
// Create and initialize the DB version table if it doesn't exist.
func ensureDBVersion(db *sql.DB) (int, error) {

	dbversion := int(0)
	row := db.QueryRow("SELECT version_id from goose_db_version ORDER BY tstamp DESC LIMIT 1;")

	if err := row.Scan(&dbversion); err == nil {
		return dbversion, nil
	}

	// if we failed, assume that the table didn't exist, and try to create it
	txn, err := db.Begin()
	if err != nil {
		return 0, err
	}

	// create the table and insert an initial value of 0
	create := `CREATE TABLE goose_db_version (
                version_id int NOT NULL,
                tstamp timestamp NULL default now(),
                PRIMARY KEY(tstamp)
              );`
	insert := "INSERT INTO goose_db_version (version_id) VALUES (0);"

	for _, str := range []string{create, insert} {
		if _, err := txn.Exec(str); err != nil {
			txn.Rollback()
			return 0, err
		}
	}

	return 0, txn.Commit()
}

// wrapper for ensureDBVersion for callers that don't already have
// their own DB instance
func getDBVersion(conf *DBConf) int {

	db, err := sql.Open(conf.Driver, conf.OpenStr)
	if err != nil {
		log.Fatal("couldn't open DB:", err)
	}
	defer db.Close()

	version, err := ensureDBVersion(db)
	if err != nil {
		log.Fatalf("couldn't get DB version: %v", err)
	}

	return version
}
