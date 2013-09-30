package main

import (
	"bitbucket.org/liamstask/goose/lib/goose"
	"log"
)

var redoCmd = &Command{
	Name:    "redo",
	Usage:   "",
	Summary: "Re-run the latest migration",
	Help:    `redo extended help here...`,
}

func redoRun(cmd *Command, args ...string) {
	conf, err := goose.NewDBConf(*flagPath, *flagEnv)
	if err != nil {
		log.Fatal(err)
	}

	current, err := goose.GetDBVersion(conf)
	if err != nil {
		log.Fatal(err)
	}

	previous, err := goose.GetPreviousDBVersion(conf.MigrationsDir, current)
	if err != nil {
		log.Fatal(err)
	}

	goose.RunMigrations(conf, conf.MigrationsDir, previous)
	goose.RunMigrations(conf, conf.MigrationsDir, current)
}

func init() {
	redoCmd.Run = redoRun
}
