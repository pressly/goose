package main

import (
	"log"
	"os"
	"path"
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

	dir, err := os.Open(dirpath)
	if err != nil {
		log.Fatal(err)
	}

	names, err := dir.Readdirnames(0)
	if err != nil {
		log.Fatal(err)
	}

	mostRecent := -1

	for _, name := range names {

		if ext := path.Ext(name); ext != ".go" && ext != ".sql" {
			continue
		}

		v, e := numericComponent(name)
		if e != nil {
			continue
		}

		if v > mostRecent {
			mostRecent = v
		}
	}

	return mostRecent
}

func init() {
	upCmd.Run = upRun
}
