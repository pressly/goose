package main

import (
	"log"
	"os"
	"path"
	"path/filepath"
)

var upCmd = &Command{
	Name:    "up",
	Usage:   "",
	Summary: "Migrate the DB to the most recent version available",
	Help:    `up extended help here...`,
}

var dbFolder = upCmd.Flag.String("db", "db", "folder containing db info")
var dbConfName = upCmd.Flag.String("config", "development", "which DB configuration to use")

func upRun(cmd *Command, args ...string) {

	conf, err := dbConfFromFile(path.Join(*dbFolder, "dbconf.yml"), *dbConfName)
	if err != nil {
		log.Fatal(err)
	}

	folder := path.Join(*dbFolder, "migrations")
	target := mostRecentVersionAvailable(folder)
	runMigrations(conf, folder, target)
}

// helper to identify the most recent possible version
// within a folder of migration scripts
func mostRecentVersionAvailable(dirpath string) int {

	mostRecent := -1

	filepath.Walk(dirpath, func(name string, info os.FileInfo, err error) error {

		if v, e := numericComponent(name); e == nil {
			if v > mostRecent {
				mostRecent = v
			}
		}

		return nil
	})

	return mostRecent
}

func init() {
	upCmd.Run = upRun
}
