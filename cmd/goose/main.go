package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pressly/goose"

	// Init DB drivers.
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/ziutek/mymysql/godrv"
)

const (
	envKeyDriver   = `GOOSE_DRIVER`
	envKeyDbstring = `GOOSE_DBSTRING`
)

var (
	flags = flag.NewFlagSet("goose", flag.ExitOnError)
	dir   = flags.String("dir", ".", "directory with migration files")
)

func configuration() (driver, dbstring, command string, arguments []string, help bool) {
	flags.Usage = usage
	flags.Parse(os.Args[1:])

	args := flags.Args()
	if args[0] == "-h" || args[0] == "--help" {
		help = true
		return
	}

	if len(args) > 1 && args[0] == "create" {
		if err := goose.Run("create", nil, *dir, args[1:]...); err != nil {
			log.Fatalf("goose run: %v", err)
		}
		return
	}

	if os.Getenv(envKeyDriver) != "" || os.Getenv(envKeyDbstring) != "" {
		return environment()
	}

	if len(args) < 3 {
		help = true
		return
	}

	driver, dbstring, command = args[0], args[1], args[2]
	if len(args) > 3 {
		arguments = append(arguments, args[3:]...)
	}

	return
}

func environment() (driver, dbstring, command string, arguments []string, help bool) {
	var args []string

	driver = os.Getenv(envKeyDriver)
	dbstring = os.Getenv(envKeyDbstring)
	args = flags.Args()
	if len(args) < 1 {
		help = true
		return
	}
	command = args[0]
	if len(args) > 1 {
		arguments = append(arguments, args[1:]...)
	}

	return
}

func main() {
	driver, dbstring, command, arguments, help := configuration()
	if help {
		flags.Usage()
		return
	} else if driver == "" {
		return
	}

	switch driver {
	case "postgres", "mysql", "sqlite3", "redshift":
		if err := goose.SetDialect(driver); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("%q driver not supported\n", driver)
	}

	switch dbstring {
	case "":
		log.Fatalf("-dbstring=%q not supported\n", dbstring)
	default:
	}

	if driver == "redshift" {
		driver = "postgres"
	}

	db, err := sql.Open(driver, dbstring)
	if err != nil {
		log.Fatalf("-dbstring=%q: %v\n", dbstring, err)
	}

	if err := goose.Run(command, db, *dir, arguments...); err != nil {
		log.Fatalf("goose run: %v", err)
	}
}

func usage() {
	fmt.Print(usagePrefix)
	flags.PrintDefaults()
	fmt.Print(usageCommands)
}

var (
	usagePrefix = `Usage: goose [OPTIONS] DRIVER DBSTRING COMMAND

or

Set environment key
GOOSE_DRIVER=DRIVER
GOOSE_DBSTRING=DBSTRING

Usage: goose [OPTIONS] COMMAND

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
    goose mysql "user:password@/dbname" status
    goose redshift "postgres://user:password@qwerty.us-east-1.redshift.amazonaws.com:5439/db" status

    GOOSE_DRIVER=sqlite3 GOOSE_DBSTRING=./foo.db goose status
    GOOSE_DRIVER=sqlite3 GOOSE_DBSTRING=./foo.db goose create init sql

    GOOSE_DRIVER=postgres GOOSE_DBSTRING="user=postgres dbname=postgres sslmode=disable" goose status
    GOOSE_DRIVER=mysql GOOSE_DBSTRING="user:password@/dbname" goose status
    GOOSE_DRIVER=redshift GOOSE_DBSTRING="postgres://user:password@qwerty.us-east-1.redshift.amazonaws.com:5439/db" goose status

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
    create NAME [sql|go] Creates new migration file with next version
`
)
