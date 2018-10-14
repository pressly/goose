package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"io/ioutil"

	"github.com/pressly/goose"
	// YAML support
	"gopkg.in/yaml.v2"

	// Init DB drivers.
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/ziutek/mymysql/godrv"
)

var (
	flags         = flag.NewFlagSet("goose", flag.ExitOnError)
	dir           = flags.String("dir", ".", "directory with migration files")
	filepath      = flags.String("file", "", "yaml file with database config file")
	environment   = flags.String("environment", "development", "database environment config")
)

type YamlData struct {
	Driver string `yaml:driver`
	DBString string `yaml:open`
}

func main() {
	flags.Usage = usage
	flags.Parse(os.Args[1:])

	args := flags.Args()

	if len(args) > 1 && args[0] == "create" {
		if err := goose.Run("create", nil, *dir, args[1:]...); err != nil {
			log.Fatalf("goose run: %v", err)
		}
		return
	}

	if args[0] == "-h" || args[0] == "--help" {
		flags.Usage()
		return
	}

	var driver string
	var dbstring string
	var command string

	if *filepath != "" {
		var database_environment string
		database_environment = *environment
		log.Printf("environment: %s", database_environment)
		file, err := ioutil.ReadFile(*filepath)
		if err != nil {
			log.Fatal(err)
		}
		tmp := make(map[interface {}]interface {})
		if err := yaml.Unmarshal([]byte(file), &tmp); err != nil {
			log.Fatal(err)
		}
		conf := tmp[database_environment].(map[interface {}]interface {})
		driver, dbstring, command = conf["driver"].(string), conf["open"].(string), args[0]
		log.Printf("%s, %s, %s",driver, dbstring, command)
	} else if len(args) < 3 {
		flags.Usage()
		return
	} else {
	        driver, dbstring, command = args[0], args[1], args[2]
	}


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

	if err := goose.Run(command, db, *dir, arguments...); err != nil {
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
