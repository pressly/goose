package main

import (
	"database/sql"
	"flag"
	"log"
	"os"

	"github.com/elijahcarrel/goose"
)

var (
	flags          = flag.NewFlagSet("goose", flag.ExitOnError)
	runParams      = goose.RunParams{
		Dir:            flags.String("dir", ".", "directory with migration files"),
		MissingOnly:    flags.Bool("missing-only", false, "for status command - show only migrations that were missed"),
		IncludeMissing: flags.Bool("include-missing", false, "for up or up-to command - include migrations that were missed"),
	}
)

func main() {
	flags.Usage = usage
	flags.Parse(os.Args[1:])

	args := flags.Args()
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		flags.Usage()
		return
	}

	switch args[0] {
	case "create":
		if err := goose.Run("create", nil, runParams, args[1:]...); err != nil {
			log.Fatalf("goose run: %v", err)
		}
		return
	case "fix":
		if err := goose.Run("fix", nil, runParams); err != nil {
			log.Fatalf("goose run: %v", err)
		}
		return
	}

	if len(args) < 3 {
		flags.Usage()
		return
	}

	driver, dbstring, command := args[0], args[1], args[2]

	if err := goose.SetDialect(driver); err != nil {
		log.Fatal(err)
	}

	switch driver {
	case "redshift":
		driver = "postgres"
	case "tidb":
		driver = "mysql"
	}

	switch dbstring {
	case "":
		log.Fatalf("-dbstring=%q not supported\n", dbstring)
	default:
	}

	db, err := sql.Open(driver, dbstring)
	if err != nil {
		log.Fatalf("-dbstring=%q: %v\n", dbstring, err)
	}

	arguments := []string{}
	if len(args) > 3 {
		arguments = append(arguments, args[3:]...)
	}

	if err := goose.Run(command, db, runParams, arguments...); err != nil {
		log.Fatalf("goose run: %v", err)
	}
}

func usage() {
	log.Print(usagePrefix)
	flags.PrintDefaults()
	log.Print(usageCommands)
}

var (
	usagePrefix = `Usage: goose [OPTIONS] DRIVER DBSTRING COMMAND

Drivers:
    postgres
    mysql
    sqlite3
    redshift

Examples:
    goose sqlite3 ./foo.db status
    goose sqlite3 ./foo.db create init sql
    goose sqlite3 ./foo.db create add_some_column sql
    goose sqlite3 ./foo.db create fetch_user_data go
    goose sqlite3 ./foo.db up

    goose postgres "user=postgres dbname=postgres sslmode=disable" status
    goose mysql "user:password@/dbname?parseTime=true" status
    goose redshift "postgres://user:password@qwerty.us-east-1.redshift.amazonaws.com:5439/db" status
    goose tidb "user:password@/dbname?parseTime=true" status

Options:
    --dir string
        directory with migration files (default ".")
    --missing-only
        for status command - show only migrations that were missed
	--include-missing
		for up or up-to command - include migrations that were missed
`

	usageCommands = `
Commands:
    up                   Migrate the DB to the most recent version available. Use [--include-missing] include migrations that were missed
    up-by-one            Migrate up by a single version
    up-to VERSION        Migrate the DB to a specific VERSION. Use [--include-missing] include migrations that were missed
    down                 Roll back the version by 1
    down-to VERSION      Roll back to a specific VERSION
    redo                 Re-run the latest migration
    reset                Roll back all migrations
    status               Dump the migration status for the current DB. Use [--missing-only] option to show only migrations that were missed
    version              Print the current version of the database
    create NAME [sql|go] Creates new migration file with the current timestamp
    fix                  Apply sequential ordering to migrations
`
)
