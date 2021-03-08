package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pressly/goose"
)

var (
	flags   = flag.NewFlagSet("migrate", flag.ExitOnError)
	env     = flags.String("env", "development", "which DB environment to use")
	dir     = flags.String("dir", "", "directory with migration files")
	verbose = flags.Bool("v", false, "enable verbose mode")
	help    = flags.Bool("h", false, "print help")
	version = flags.Bool("version", false, "print version")
)

func main() {
	flags.Usage = usage
	if err := flags.Parse(os.Args[1:]); err != nil {
		log.Panic(err)
	}

	if *version {
		fmt.Println(goose.VERSION)
		return
	}
	if *verbose {
		goose.SetVerbose(true)
	}

	args := flags.Args()
	if len(args) == 0 || *help {
		flags.Usage()
		return
	}

	conf, err := NewDBConf(*dir, *env, "")
	if err != nil {
		log.Fatalf("unable to process dbconf.yml: %v\n", err)
	}

	switch args[0] {
	case "create":
		if err := goose.Run("create", nil, conf.MigrationsDir, args[1:]...); err != nil {
			log.Fatalf("migrate create: %s", err)
		}
		return
	case "fix":
		if err := goose.Run("fix", nil, conf.MigrationsDir); err != nil {
			log.Fatalf("migrate fix: %s", err)
		}
		return
	}

	if len(args) < 1 {
		flags.Usage()
		return
	}

	db, err := goose.OpenDBWithDriver(conf.Driver, conf.DBString)
	if err != nil {
		log.Fatalf("dbstring=%q: %v\n", conf.DBString, err)
	}

	arguments := []string{}
	if len(args) > 1 {
		arguments = append(arguments, args[3:]...)
	}

	if err := goose.Run(args[0], db, conf.MigrationsDir, arguments...); err != nil {
		log.Fatalf("goose %s: %s", args[0], err)
	}
}

func usage() {
	fmt.Println(usagePrefix)
	flags.PrintDefaults()
	fmt.Println(usageCommands)
}

var (
	usagePrefix = `Usage: migrate [OPTIONS] COMMAND
Examples:
    migrate status
    migrate create init sql
    migrate create add_some_column sql
    migrate create fetch_user_data go
    migrate up
Options:
`

	usageCommands = `
Commands:
    up                   Migrate the DB to the most recent version available
    up-to VERSION        Migrate the DB to a specific VERSION
    down                 Roll back the version by 1
    down-to VERSION      Roll back to a specific VERSION
    redo                 Re-run the latest migration
    reset                Roll back all migrations
    status               Dump the migration status for the current DB
    version              Print the current version of the database
    create NAME [sql|go] Creates new migration file with the current timestamp
    fix                  Apply sequential ordering to migrations
`
)
