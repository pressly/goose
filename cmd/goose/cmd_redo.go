package main

import "log"

var redoCmd = &Command{
	Name:    "redo",
	Usage:   "",
	Summary: "Re-run the latest migration",
	Help:    `redo extended help here...`,
}

func redoRun(cmd *Command, args ...string) {
	conf, err := NewDBConf()
	if err != nil {
		log.Fatal(err)
	}

	target := getDBVersion(conf)
	_, earliest := getPreviousVersion(conf.MigrationsDir, target)

	downRun(cmd, args...)
	if target == 0 {
		log.Printf("Updating from %s to %s\n", target, earliest)
		target = earliest
	}
	runMigrations(conf, conf.MigrationsDir, target)
}

func init() {
	redoCmd.Run = redoRun
}
