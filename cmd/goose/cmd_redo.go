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

	target, err := goose.GetDBVersion(conf)
	if err != nil {
		log.Fatal(err)
	}

	_, earliest := goose.GetPreviousDBVersion(conf.MigrationsDir, target)

	downRun(cmd, args...)
	if target == 0 {
		log.Printf("Updating from %s to %s\n", target, earliest)
		target = earliest
	}
	goose.RunMigrations(conf, conf.MigrationsDir, target)
}

func init() {
	redoCmd.Run = redoRun
}
