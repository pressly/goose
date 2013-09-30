package main

import (
	"bitbucket.org/liamstask/goose/lib/goose"
	"fmt"
	"log"
)

var downCmd = &Command{
	Name:    "down",
	Usage:   "",
	Summary: "Roll back the version by 1",
	Help:    `down extended help here...`,
}

func downRun(cmd *Command, args ...string) {

	conf, err := goose.NewDBConf(*flagPath, *flagEnv)
	if err != nil {
		log.Fatal(err)
	}

	current := goose.GetDBVersion(conf)
	if current == 0 {
		fmt.Println("db is empty, can't go down.")
		return
	}

	previous, earliest := goose.GetPreviousDBVersion(conf.MigrationsDir, current)

	// if we're at the earliest version, indicate that the
	// only available step is to roll back to an empty database
	if current == earliest {
		previous = 0
	}

	goose.RunMigrations(conf, conf.MigrationsDir, previous)
}

func init() {
	downCmd.Run = downRun
}
