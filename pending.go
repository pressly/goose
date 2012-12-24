package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
)

var pendingCmd = &Command{
	Name:    "pending",
	Usage:   "",
	Summary: "Display any migrations yet to be applied",
	Help:    `pending extended help here...`,
}

// entry point for the `goose pending` Command
func pendingRun(cmd *Command, args ...string) {

	conf, e := MakeDBConf()
	if e != nil {
		log.Fatal("config error:", e)
	}

	current := getDBVersion(conf)
	pendingScripts := collectPendingMigrations(conf.MigrationsDir, current)

	if len(pendingScripts) == 0 {
		fmt.Printf("goose: no pending migrations. you're up to date at version %v\n", current)
	} else {
		fmt.Printf("goose: %v pending migration(s):\n", len(pendingScripts))
		for _, s := range pendingScripts {
			fmt.Printf("    %v\n", s)
		}
	}
}

// collect all migrations that specify a version later than our current version
func collectPendingMigrations(dirpath string, current int) []string {

	var pendingScripts []string

	// XXX: would be much better to query the DB for applied versions.

	filepath.Walk(dirpath, func(name string, info os.FileInfo, err error) error {

		if v, e := numericComponent(name); e == nil {
			if v > current {
				pendingScripts = append(pendingScripts, path.Base(name))
			}
		}

		return nil
	})

	return pendingScripts
}

func init() {
	pendingCmd.Run = pendingRun
}
