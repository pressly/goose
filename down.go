package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
)

var downCmd = &Command{
	Name:    "down",
	Usage:   "",
	Summary: "Roll back the version by 1",
	Help:    `down extended help here...`,
}

var downDBFolder = downCmd.Flag.String("db", "db", "folder containing db info")
var downDBConfName = downCmd.Flag.String("config", "development", "which DB configuration to use")

func downRun(cmd *Command, args ...string) {

	conf, err := dbConfFromFile(path.Join(*downDBFolder, "dbconf.yml"), *downDBConfName)
	if err != nil {
		log.Fatal(err)
	}

	current := getDBVersion(conf)
	folder := path.Join(*downDBFolder, "migrations")
	previous, earliest := getPreviousVersion(folder, current)

	if current == 0 {
		fmt.Println("db is empty, can't go down.")
		return
	}

	// if we're at the earliest version, indicate that the
	// only available step is to roll back to an empty database
	if current == earliest {
		previous = 0
	}

	runMigrations(conf, folder, previous)
}

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

func getPreviousVersion(dirpath string, version int) (previous, earliest int) {

	previous = -1
	earliest = (1 << 31) - 1

	filepath.Walk(dirpath, func(name string, info os.FileInfo, err error) error {

		if v, e := numericComponent(name); e == nil {
			if v > previous && v < version {
				previous = v
			}

			if v < earliest {
				earliest = v
			}
		}

		return nil
	})

	return previous, earliest
}

func init() {
	downCmd.Run = downRun
}
