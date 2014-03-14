package main

import (
	"bitbucket.org/liamstask/goose/lib/goose"
	"fmt"
	"log"
)

var versionCmd = &Command{
	Name:    "version",
	Usage:   "",
	Summary: "Retrieve the current version for the DB",
	Help:    `version extended help here...`,
	Run:     versionRun,
}

func versionRun(cmd *Command, args ...string) {
	conf, err := dbConfFromFlags()
	if err != nil {
		log.Fatal(err)
	}

	current, err := goose.GetDBVersion(conf)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%v\n", current)
}
