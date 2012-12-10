package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	_ "github.com/bmizerany/pq"
	"github.com/kylelemons/go-gypsy/yaml"
	"log"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

type DBConf struct {
	Name    string
	Driver  string
	OpenStr string
}

type DBVersion struct {
	VersionId int
	TStamp    time.Time
}

type MigrationMap struct {
	Versions  []int          // sorted slice of version keys
	Sources   map[int]string // sources (.sql or .go) keyed by version
	Direction bool           // sort direction: true -> Up, false -> Down
}

var dbFolder = flag.String("db", "db", "folder containing db info")
var dbConfName = flag.String("config", "development", "which DB configuration to use")
var targetVersion = flag.Int("target", -1, "which DB version to target (defaults to latest version)")

func main() {
	flag.Parse()

	conf, err := dbConfFromFile(path.Join(*dbFolder, "dbconf.yaml"), *dbConfName)
	if err != nil {
		log.Fatal(err)
	}

	runMigrations(conf, *targetVersion)
}

// extract configuration details from the given file
func dbConfFromFile(path, envtype string) (*DBConf, error) {

	f, err := yaml.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	drv, derr := f.Get(fmt.Sprintf("%s.driver", envtype))
	if derr != nil {
		return nil, derr
	}

	open, oerr := f.Get(fmt.Sprintf("%s.open", envtype))
	if oerr != nil {
		return nil, oerr
	}

	return &DBConf{
		Name:    envtype,
		Driver:  drv,
		OpenStr: open,
	}, nil
}

func runMigrations(conf *DBConf, target int) {

	db, err := sql.Open(conf.Driver, conf.OpenStr)
	if err != nil {
		log.Fatal("couldn't open DB:", err)
	}

	currentVersion, e := ensureDBVersion(db)
	if e != nil {
		log.Fatal("couldn't get/set DB version")
	}

	migrations, err := collectMigrations(path.Join(*dbFolder, "migrations"), currentVersion)
	if err != nil {
		log.Fatal(err)
	}

	if len(migrations.Versions) == 0 {
		fmt.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
		return
	}

	fmt.Printf("goose: migrating db configuration '%v', current version: %d, target: %d\n",
		conf.Name, currentVersion, *targetVersion)

	for _, v := range migrations.Versions {

		txn, err := db.Begin()
		if err != nil {
			log.Fatal("db.Begin:", err)
		}

		var numStatements int
		var e error

		filepath := migrations.Sources[v]

		switch path.Ext(filepath) {
		case ".go":
			numStatements, e = runGoMigration(txn, conf, filepath, v, migrations.Direction)
		case ".sql":
			numStatements, e = runSQLMigration(txn, filepath, v, migrations.Direction)
		}

		if e != nil {
			txn.Rollback()
			fmt.Printf("FAIL %s (%v), quitting migration.", path.Base(filepath), e)
			os.Exit(1)
		}

		if e = finalizeMigration(txn, migrations, v); e != nil {
			fmt.Printf("error finalizing migration %s, quitting. (%v)", path.Base(filepath), e)
			os.Exit(1)
		}

		fmt.Printf("OK   %s (%d statements)\n", path.Base(filepath), numStatements)
	}
}

// Update the version table for the given migration,
// and finalize the transaction.
func finalizeMigration(txn *sql.Tx, mm *MigrationMap, v int) error {

	if mm.Direction == false {
		v--
	}

	// XXX: drop goose_db_version table on some minimum version number?
	versionStmt := fmt.Sprintf("INSERT INTO goose_db_version (version_id) VALUES (%d);", v)
	if _, err := txn.Exec(versionStmt); err != nil {
		txn.Rollback()
		return err
	}

	return txn.Commit()
}

// collect all the valid looking migration scripts in the 
// migrations folder, and key them by version
func collectMigrations(dirpath string, currentVersion int) (mm *MigrationMap, err error) {

	dir, err := os.Open(dirpath)
	if err != nil {
		log.Fatal(err)
	}

	names, err := dir.Readdirnames(0)
	if err != nil {
		log.Fatal(err)
	}

	mm = &MigrationMap{
		Sources: make(map[int]string),
	}

	// extract the numeric component of each migration,
	// filter out any uninteresting files,
	// and ensure we only have one file per migration version.
	for _, name := range names {

		ext := path.Ext(name)
		if ext != ".go" && ext != ".sql" {
			continue
		}

		v, e := numericComponent(name)
		if e != nil {
			continue
		}

		if _, ok := mm.Sources[v]; ok {
			log.Fatalf("more than one file specifies the migration for version %d (%s and %s)",
				v, mm.Versions[v], path.Join(dirpath, name))
		}

		if versionFilter(v, currentVersion, *targetVersion) {
			mm.Append(v, path.Join(dirpath, name))
		}
	}

	if len(mm.Versions) > 0 {
		mm.Sort(currentVersion)
	}

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
		return v <= current && v >= target
	}

	return false
}

func (m *MigrationMap) Append(v int, source string) {
	m.Versions = append(m.Versions, v)
	m.Sources[v] = source
}

func (m *MigrationMap) Sort(currentVersion int) {
	sort.Ints(m.Versions)

	// side effect - default to max version?
	if *targetVersion < 0 {
		*targetVersion = m.Versions[len(m.Versions)-1]
	}

	// set direction, and reverse order if need be
	m.Direction = currentVersion < *targetVersion
	if m.Direction == false {
		for i, j := 0, len(m.Versions)-1; i < j; i, j = i+1, j-1 {
			m.Versions[i], m.Versions[j] = m.Versions[j], m.Versions[i]
		}
	}
}

// look for migration scripts with names in the form:
//  XXX_descriptivename.ext
// where XXX specifies the version number
// and ext specifies the type of migration
func numericComponent(path string) (int, error) {
	idx := strings.Index(path, "_")
	if idx < 0 {
		return 0, errors.New("no separator found")
	}
	return strconv.Atoi(path[:idx])
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

	for _, str := range [2]string{create, insert} {
		if _, err := txn.Exec(str); err != nil {
			txn.Rollback()
			return 0, err
		}
	}

	return 0, txn.Commit()
}
