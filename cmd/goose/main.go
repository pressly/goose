package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"runtime/debug"
	"strconv"
	"text/template"

	"github.com/pressly/goose/v4"
)

var (
	flags        = flag.NewFlagSet("goose", flag.ExitOnError)
	dir          = flags.String("dir", defaultMigrationDir, "directory with migration files")
	table        = flags.String("table", "goose_db_version", "migrations table name")
	verbose      = flags.Bool("v", false, "enable verbose mode")
	help         = flags.Bool("h", false, "print help")
	version      = flags.Bool("version", false, "print version")
	sequential   = flags.Bool("s", false, "use sequential numbering for new migrations")
	allowMissing = flags.Bool("allow-missing", false, "applies missing (out-of-order) migrations")
	noVersioning = flags.Bool("no-versioning", false, "apply migration commands with no versioning, in file order, from directory pointed to")

	certfile = flags.String("certfile", "", "file path to root CA's certificates in pem format (only support on mysql)")
	sslcert  = flags.String("ssl-cert", "", "file path to SSL certificates in pem format (only support on mysql)")
	sslkey   = flags.String("ssl-key", "", "file path to SSL key in pem format (only support on mysql)")
)
var (
	gooseVersion = ""
)

func main() {
	flags.Usage = usage
	flags.Parse(os.Args[1:])

	if *version {
		if buildInfo, ok := debug.ReadBuildInfo(); ok && buildInfo != nil && gooseVersion == "" {
			gooseVersion = buildInfo.Main.Version
		}
		fmt.Printf("goose version:%s\n", gooseVersion)
		return
	}

	args := flags.Args()
	if len(args) == 0 || *help {
		flags.Usage()
		return
	}
	// The -dir option has not been set, check whether the env variable is set
	// before defaulting to ".".
	if *dir == defaultMigrationDir && os.Getenv(envGooseMigrationDir) != "" {
		*dir = os.Getenv(envGooseMigrationDir)
	}

	switch args[0] {
	case "init":
		filename, err := gooseInit(*dir)
		if err != nil {
			log.Fatalf("goose run: %v", err)
		}
		log.Printf("created new file: %s\n", filename)
		return
	case "create":
		_ = *sequential
		// if err := goose.Run("create", nil, *dir, args[1:]...); err != nil {
		// 	log.Fatalf("goose run: %v", err)
		// }
		return
	case "fix":
		// if err := goose.Run("fix", nil, *dir); err != nil {
		// 	log.Fatalf("goose run: %v", err)
		// }
		return
	}

	args = mergeArgs(args)
	if len(args) < 3 {
		flags.Usage()
		return
	}

	driver, dbstring, command := args[0], args[1], args[2]
	// To avoid breaking existing consumers, treat sqlite3 as sqlite.
	// An implementation detail that consumers should not care which
	// underlying driver is used. Internally uses the CGo-free port
	// of SQLite: modernc.org/sqlite
	if driver == "sqlite3" {
		driver = "sqlite"
	}
	dialect := goose.DialectPostgres
	db, err := sql.Open("postgres", dbstring)
	if err != nil {
		log.Fatal(err)
	}
	options := &goose.Options{
		TableName:    *table,
		AllowMissing: *allowMissing,
		NoVersioning: *noVersioning,
		Verbose:      *verbose,
	}
	provider, err := goose.NewProvider(dialect, db, *dir, options)
	if err != nil {
		log.Fatalf("-dbstring=%q: %v\n", dbstring, err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Fatalf("goose: failed to close DB: %v\n", err)
		}
	}()

	arguments := []string{}
	if len(args) > 3 {
		arguments = append(arguments, args[3:]...)
	}

	if err := run(
		command,
		provider,
		*dir,
		arguments,
	); err != nil {
		log.Fatalf("goose run: %v", err)
	}
}

const (
	envGooseDriver       = "GOOSE_DRIVER"
	envGooseDBString     = "GOOSE_DBSTRING"
	envGooseMigrationDir = "GOOSE_MIGRATION_DIR"
)

const (
	defaultMigrationDir = "."
)

func mergeArgs(args []string) []string {
	if len(args) < 1 {
		return args
	}
	if d := os.Getenv(envGooseDriver); d != "" {
		args = append([]string{d}, args...)
	}
	if d := os.Getenv(envGooseDBString); d != "" {
		args = append([]string{args[0], d}, args[1:]...)
	}
	return args
}

func usage() {
	fmt.Println(usagePrefix)
	flags.PrintDefaults()
	fmt.Println(usageCommands)
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
    mssql
    redshift
    tidb
    clickhouse

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
    goose mssql "sqlserver://user:password@dbname:1433?database=master" status
    goose clickhouse "tcp://127.0.0.1:9000" status

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
    up-by-one            Migrate the DB up by 1
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

var sqlMigrationTemplate = template.Must(template.New("goose.sql-migration").Parse(`-- Thank you for giving goose a try!
-- 
-- This file was automatically created running goose init. If you're familiar with goose
-- feel free to remove/rename this file, write some SQL and goose up. Briefly,
-- 
-- Documentation can be found here: https://pressly.github.io/goose
--
-- A single goose .sql file holds both Up and Down migrations.
-- 
-- All goose .sql files are expected to have a -- +goose Up directive.
-- The -- +goose Down directive is optional, but recommended, and must come after the Up directive.
-- 
-- The -- +goose NO TRANSACTION directive may be added to the top of the file to run statements 
-- outside a transaction. Both Up and Down migrations within this file will be run without a transaction.
-- 
-- More complex statements that have semicolons within them must be annotated with 
-- the -- +goose StatementBegin and -- +goose StatementEnd directives to be properly recognized.
-- 
-- Use GitHub issues for reporting bugs and requesting features, enjoy!

-- +goose Up
SELECT 'up SQL query';

-- +goose Down
SELECT 'down SQL query';
`))

// initDir will create a directory with an empty SQL migration file.
func gooseInit(dir string) (string, error) {
	if dir == "" || dir == defaultMigrationDir {
		dir = "migrations"
	}
	_, err := os.Stat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err == nil, errors.Is(err, fs.ErrExist):
		return "", fmt.Errorf("directory already exists: %s", dir)
	default:
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return goose.CreateWithTemplate(dir, sqlMigrationTemplate, "initial", "sql", true)
}

func run(command string, p *goose.Provider, dir string, args []string) error {
	ctx := context.Background()
	switch command {
	case "up":
		if err := p.Up(ctx); err != nil {
			return err
		}
	case "up-by-one":
		if err := p.UpByOne(ctx); err != nil {
			return err
		}
	case "up-to":
		if len(args) == 0 {
			return fmt.Errorf("up-to must be of form: goose [OPTIONS] DRIVER DBSTRING up-to VERSION")
		}
		version, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("version must be a number (got '%s')", args[0])
		}
		if err := p.UpTo(ctx, version); err != nil {
			return err
		}
	case "create":
		// if len(args) == 0 {
		// 	return fmt.Errorf("create must be of form: goose [OPTIONS] DRIVER DBSTRING create NAME [go|sql]")
		// }

		// migrationType := "go"
		// if len(args) == 2 {
		// 	migrationType = args[1]
		// }
		// if err := goose.Create(p.D, dir, args[0], migrationType); err != nil {
		// 	return err
		// }
	case "down":
		if err := p.Down(ctx); err != nil {
			return err
		}
	case "down-to":
		if len(args) == 0 {
			return fmt.Errorf("down-to must be of form: goose [OPTIONS] DRIVER DBSTRING down-to VERSION")
		}
		version, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("version must be a number (got '%s')", args[0])
		}
		if err := p.DownTo(ctx, version); err != nil {
			return err
		}
	case "fix":
		// if err := Fix(dir); err != nil {
		// 	return err
		// }
	case "redo":
		if err := p.Redo(ctx); err != nil {
			return err
		}
	case "reset":
		if err := p.Reset(ctx); err != nil {
			return err
		}
	case "status":
		// TODO(mf): implement
	case "version":
		currentVersion, err := p.CurrentVersion(ctx)
		if err != nil {
			return err
		}
		log.Printf("goose: version %v\n", currentVersion)
	default:
		return fmt.Errorf("%q: no such command", command)
	}
	return nil
}
