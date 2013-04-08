package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var downCmd = &Command{
	Name:    "down",
	Usage:   "",
	Summary: "Roll back the version by 1",
	Help:    `down extended help here...`,
}

func downRun(cmd *Command, args ...string) {

	conf, err := NewDBConf()
	if err != nil {
		log.Fatal(err)
	}

	current := getDBVersion(conf)
	previous, earliest := getPreviousVersion(conf.MigrationsDir, current)

	if current == 0 {
		fmt.Println("db is empty, can't go down.")
		return
	}

	// if we're at the earliest version, indicate that the
	// only available step is to roll back to an empty database
	if current == earliest {
		previous = 0
	}

	runMigrations(conf, conf.MigrationsDir, previous)
}

func getPreviousVersion(dirpath string, version int64) (previous, earliest int64) {

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
