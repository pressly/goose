package main

import (
	"bitbucket.org/liamstask/goose/lib/goose"
	"log"
)

var upCmd = &Command{
	Name:    "up",
	Usage:   "",
	Summary: "Migrate the DB to the most recent version available",
	Help:    `up extended help here...`,
}

func upRun(cmd *Command, args ...string) {

	conf, err := goose.NewDBConf(*flagPath, *flagEnv)
	if err != nil {
		log.Fatal(err)
	}

	target, err := goose.GetMostRecentDBVersion(conf.MigrationsDir)
	if err != nil {
		log.Fatal(err)
	}

	if err := goose.RunMigrations(conf, conf.MigrationsDir, target); err != nil {
		log.Fatal(err)
	}
}

func init() {
	upCmd.Run = upRun
}
