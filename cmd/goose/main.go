package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/cfg"
	"github.com/pressly/goose/v3/internal/migrationstats"
	"github.com/pressly/goose/v3/internal/migrationstats/migrationstatsos"
)

var (
	flags        = flag.NewFlagSet("goose", flag.ExitOnError)
	dir          = flags.String("dir", cfg.DefaultMigrationDir, "directory with migration files")
	table        = flags.String("table", "goose_db_version", "migrations table name")
	verbose      = flags.Bool("v", false, "enable verbose mode")
	help         = flags.Bool("h", false, "print help")
	version      = flags.Bool("version", false, "print version")
	certfile     = flags.String("certfile", "", "file path to root CA's certificates in pem format (only support on mysql)")
	sequential   = flags.Bool("s", false, "use sequential numbering for new migrations")
	allowMissing = flags.Bool("allow-missing", false, "applies missing (out-of-order) migrations")
	sslcert      = flags.String("ssl-cert", "", "file path to SSL certificates in pem format (only support on mysql)")
	sslkey       = flags.String("ssl-key", "", "file path to SSL key in pem format (only support on mysql)")
	noVersioning = flags.Bool("no-versioning", false, "apply migration commands with no versioning, in file order, from directory pointed to")
	noColor      = flags.Bool("no-color", false, "disable color output (NO_COLOR env variable supported)")
)
var (
	gooseVersion = ""
)

func main() {
	flags.Usage = usage
	if err := flags.Parse(os.Args[1:]); err != nil {
		log.Fatalf("failed to parse args: %v", err)
		return
	}

	if *version {
		if buildInfo, ok := debug.ReadBuildInfo(); ok && buildInfo != nil && gooseVersion == "" {
			gooseVersion = buildInfo.Main.Version
		}
		fmt.Printf("goose version:%s\n", gooseVersion)
		return
	}
	if *verbose {
		goose.SetVerbose(true)
	}
	if *sequential {
		goose.SetSequential(true)
	}
	goose.SetTableName(*table)

	args := flags.Args()

	if *help {
		flags.Usage()
		return
	}

	if len(args) == 0 {
		flags.Usage()
		os.Exit(1)
	}

	// The -dir option has not been set, check whether the env variable is set
	// before defaulting to ".".
	if *dir == cfg.DefaultMigrationDir && cfg.GOOSEMIGRATIONDIR != "" {
		*dir = cfg.GOOSEMIGRATIONDIR
	}

	switch args[0] {
	case "init":
		if err := gooseInit(*dir); err != nil {
			log.Fatalf("goose run: %v", err)
		}
		return
	case "create":
		if err := goose.Run("create", nil, *dir, args[1:]...); err != nil {
			log.Fatalf("goose run: %v", err)
		}
		return
	case "fix":
		if err := goose.Run("fix", nil, *dir); err != nil {
			log.Fatalf("goose run: %v", err)
		}
		return
	case "env":
		for _, env := range cfg.List() {
			fmt.Printf("%s=%q\n", env.Name, env.Value)
		}
		return
	case "validate":
		if err := printValidate(*dir, *verbose); err != nil {
			log.Fatalf("goose validate: %v", err)
		}
		return
	}

	args = mergeArgs(args)
	if len(args) < 3 {
		flags.Usage()
		os.Exit(1)
	}

	driver, dbstring, command := args[0], args[1], args[2]
	// To avoid breaking existing consumers. An implementation detail
	// that consumers should not care which underlying driver is used.
	switch driver {
	case "sqlite3":
		//  Internally uses the CGo-free port of SQLite: modernc.org/sqlite
		driver = "sqlite"
	case "postgres":
		driver = "pgx"
	}
	db, err := goose.OpenDBWithDriver(driver, normalizeDBString(driver, dbstring, *certfile, *sslcert, *sslkey))
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
	options := []goose.OptionsFunc{}
	if *noColor || checkNoColorFromEnv() {
		options = append(options, goose.WithNoColor(true))
	}
	if *allowMissing {
		options = append(options, goose.WithAllowMissing())
	}
	if *noVersioning {
		options = append(options, goose.WithNoVersioning())
	}
	if err := goose.RunWithOptions(
		command,
		db,
		*dir,
		arguments,
		options...,
	); err != nil {
		log.Fatalf("goose run: %v", err)
	}
}

func checkNoColorFromEnv() bool {
	ok, _ := strconv.ParseBool(cfg.GOOSENOCOLOR)
	return ok
}

func mergeArgs(args []string) []string {
	if len(args) < 1 {
		return args
	}
	if s := cfg.GOOSEDRIVER; s != "" {
		args = append([]string{s}, args...)
	}
	if s := cfg.GOOSEDBSTRING; s != "" {
		args = append([]string{args[0], s}, args[1:]...)
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
    vertica

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
    goose vertica "vertica://user:password@localhost:5433/dbname?connection_load_balance=1" status

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
-- All goose .sql files are expected to have a -- +goose Up annotation.
-- The -- +goose Down annotation is optional, but recommended, and must come after the Up annotation.
-- 
-- The -- +goose NO TRANSACTION annotation may be added to the top of the file to run statements 
-- outside a transaction. Both Up and Down migrations within this file will be run without a transaction.
-- 
-- More complex statements that have semicolons within them must be annotated with 
-- the -- +goose StatementBegin and -- +goose StatementEnd annotations to be properly recognized.
-- 
-- Use GitHub issues for reporting bugs and requesting features, enjoy!

-- +goose Up
SELECT 'up SQL query';

-- +goose Down
SELECT 'down SQL query';
`))

// initDir will create a directory with an empty SQL migration file.
func gooseInit(dir string) error {
	if dir == "" || dir == cfg.DefaultMigrationDir {
		dir = "migrations"
	}
	_, err := os.Stat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err == nil, errors.Is(err, fs.ErrExist):
		return fmt.Errorf("directory already exists: %s", dir)
	default:
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return goose.CreateWithTemplate(nil, dir, sqlMigrationTemplate, "initial", "sql")
}

func gatherFilenames(filename string) ([]string, error) {
	stat, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	var filenames []string
	if stat.IsDir() {
		for _, pattern := range []string{"*.sql", "*.go"} {
			file, err := filepath.Glob(filepath.Join(filename, pattern))
			if err != nil {
				return nil, err
			}
			filenames = append(filenames, file...)
		}
	} else {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)
	return filenames, nil
}

func printValidate(filename string, verbose bool) error {
	filenames, err := gatherFilenames(filename)
	if err != nil {
		return err
	}
	fileWalker := migrationstatsos.NewFileWalker(filenames...)
	stats, err := migrationstats.GatherStats(fileWalker, false)
	if err != nil {
		return err
	}
	// TODO(mf): we should introduce a --debug flag, which allows printing
	// more internal debug information and leave verbose for additional information.
	if !verbose {
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
	fmtPattern := "%v\t%v\t%v\t%v\t%v\t\n"
	fmt.Fprintf(w, fmtPattern, "Type", "Txn", "Up", "Down", "Name")
	fmt.Fprintf(w, fmtPattern, "────", "───", "──", "────", "────")
	for _, m := range stats {
		txnStr := "✔"
		if !m.Tx {
			txnStr = "✘"
		}
		fmt.Fprintf(w, fmtPattern,
			strings.TrimPrefix(filepath.Ext(m.FileName), "."),
			txnStr,
			m.UpCount,
			m.DownCount,
			filepath.Base(m.FileName),
		)
	}
	return w.Flush()
}
