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

	target := goose.GetMostRecentDBVersion(conf.MigrationsDir)
	goose.RunMigrations(conf, conf.MigrationsDir, target)
}

func init() {
	upCmd.Run = upRun
}
