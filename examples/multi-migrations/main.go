package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pressly/goose/v3"

	_ "modernc.org/sqlite"

	"github.com/pressly/goose/v3/examples/multi-migrations/belle"
	"github.com/pressly/goose/v3/examples/multi-migrations/pardi"
)

var (
	dir          = flag.String("dir", dirPath(), "directory which hold the migration directories for pardi, belle and tests")
	dbConnection = flag.String("connection", "database.db", "database connection string")
	test         = flag.String("test", "", "create test migrations for given test dir in testdata/migrations/")
	forPardi     = flag.Bool("pardi", false, "to create files for pardi set this flag to true, does not effect fix")
)

// dirPath finds the path where our migrations live, we assume it's the directories above
// where we are
func dirPath() string {
	_, filename, _, _ := runtime.Caller(1)
	return filepath.Dir(filename)
}

func testProvider(testDir string) (*goose.Provider, string) {
	testPath := filepath.Join(*dir, "testdata", "migrations", testDir)
	// insure test dir exits
	_ = os.MkdirAll(testPath, os.ModePerm)
	// Only supports sql files
	p := goose.NewProvider(
		goose.Tablename("test_"+testDir+"_db_version"),
		goose.Dialect(goose.DialectSQLite3),
	)
	return p, testPath
}
func getAllTests() []string {
	testPath := filepath.Join(*dir, "testdata", "migrations")
	entries, err := os.ReadDir(testPath)
	if err != nil {
		log.Fatalf("Failed to read test dir: %v :%v", testPath, err)
	}
	testDirs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		testDirs = append(testDirs, entry.Name())
	}
	return testDirs
}

func main() {
	flag.Parse()
	args := flag.Args()
	command := "status"

	if len(args) >= 1 {
		command = args[0]
		args = args[1:]
	}
	db, err := sql.Open("sqlite", *dbConnection)
	if err != nil {
		log.Fatalf("%v: failed to open db: %v ", os.Args[0], err)
	}
	p := belle.Provider
	who := "belle"
	if *forPardi {
		p = pardi.Provider
		who = "pardi"
	}

	switch command {
	case "up":
		err := p.Up(db, ".")
		if err != nil {
			log.Fatalf("%s error: %v", who, err)
		}
		if *test != "" {
			p, path := testProvider(*test)
			err = p.Up(db, path)
			if err != nil {
				log.Fatalf("test %s error: %v", *test, err)
			}
		}
	case "fix":
		// first fix pardi then belle, then all the test dirs
		log.Printf("fixing pardi migration files:\n")
		err := pardi.Provider.Fix(".")
		if err != nil {
			log.Fatalf("pardi error: %v", err)
		}
		log.Printf("fixing belle migration files:\n")
		err = belle.Provider.Fix(".")
		if err != nil {
			log.Fatalf("belle error: %v", err)
		}
		for _, dir := range getAllTests() {
			log.Printf("fixing test %v migration files:\n", dir)
			p, path := testProvider(dir)
			err = p.Fix(path)
			if err != nil {
				log.Fatalf("test %v error: %v", dir, err)
			}
		}

	case "status":
		log.Printf("For %s migration:\n", who)
		err = p.Status(db, ".")
		if err != nil {
			log.Printf("Failed to get status for %s: %s", who, err)
		}
		if *test != "" {
			// output the status of the test dir
			// assume test dir always exists to keep example simple
			log.Printf("For test migration: %v\n", *test)
			p, path := testProvider(*test)
			err = p.Status(db, path)
			if err != nil {
				log.Printf("Failed to get status for test %s: %s", *test, err)
			}
		}
	case "create":
		tplType := "sql"
		if len(args) == 0 {
			log.Fatalf("Create requires at lease the name")
		}
		if len(args) == 2 {
			tplType = args[1]
		}
		if *test != "" {
			if tplType == "go" {
				log.Fatalf("test migrations files can only be sql")
			}
			p, path := testProvider(*test)
			err = p.Create(db, path, args[0], "sql")
			if err != nil {
				log.Fatalf("test %s create error: %v", *test, err)
			}
			// if test only create test files
			return
		}

		err = p.Create(db, ".", args[0], tplType)
		if err != nil {
			log.Fatalf("%s create error: %v", who, err)
		}

	}
}
