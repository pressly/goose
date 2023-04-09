package cli

import (
	"context"
	"flag"
	"io"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func newRootCmd(w io.Writer) (*ffcli.Command, *rootConfig) {
	config := &rootConfig{
		stdout: w,
	}
	fs := flag.NewFlagSet("goose", flag.ExitOnError)
	config.registerFlags(fs)

	root := &ffcli.Command{
		Name:    "goose [flags] <command> [flags] [args...]",
		FlagSet: fs,
		Options: []ff.Option{
			ff.WithEnvVarPrefix("GOOSE"),
		},
		UsageFunc: func(c *ffcli.Command) string {
			return rootUsage
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
	return root, config
}

type rootConfig struct {
	dir     string
	verbose bool
	useJSON bool

	dbstring         string
	table            string
	noVersioning     bool
	allowMissing     bool
	lockMode         string
	excludeFilenames stringSet

	// stdout is the output stream for the command. It is set to os.Stdout by default, but can be
	// overridden for testing.
	stdout io.Writer
}

// registerFlags registers the flag fields into the provided flag.FlagSet. This helper function
// allows subcommands to register the root flags into their flagsets, creating "global" flags that
// can be passed after any subcommand at the commandline.
func (c *rootConfig) registerFlags(fs *flag.FlagSet) {
	fs.BoolVar(&c.verbose, "v", false, "log verbose output")
	fs.BoolVar(&c.useJSON, "json", false, "log output as JSON")
	// Migration configuration
	fs.StringVar(&c.dir, "dir", DefaultDir, "directory with migration files")
	// Database configuration
	fs.StringVar(&c.dbstring, "dbstring", "", "database connection string")
	fs.StringVar(&c.table, "table", "goose_db_version", "database table to store version info")
	// Features
	fs.BoolVar(&c.noVersioning, "no-versioning", false, "do not use versioning")
	fs.BoolVar(&c.allowMissing, "allow-missing", false, "allow missing (out-of-order) migrations")
	fs.StringVar(&c.lockMode, "lock-mode", "", "lock mode (none, advisory-session)")
	fs.Var(&c.excludeFilenames, "exclude", "exclude filenames (comma separated)")
}

const (
	rootUsage = `
A database migration tool that simplifies the process of versioning, applying, and rolling back
schema changes in a controlled and organized way.

USAGE
  goose <command> [flags] [args...]

COMMANDS
  create          Create a new .go or .sql migration file
  down            Migrate database down to the previous version
  down-to         Migrate database down to, but not including, a specific version
  env             Print environment variables
  fix             Apply sequential numbering to existing timestamped migrations
  redo            Roll back the last appied migration and re-apply it
  status          List applied and pending migrations
  up              Migrate database to the most recent version
  up-by-one       Migrate database up by one version
  up-to           Migrate database up to, and including, a specific version
  validate        Validate migration files in the migrations directory
  version         Print the current version of the database

SUPPORTED DATABASES
  postgres        mysql        sqlite3
  redshift        tidb         mssql
  clickhouse      vertica      

FLAGS
  --allow-missing         Allow missing (out-of-order) migrations
  --dbstring              Database connection string
  --dir                   Directory with migration files (default: "./migrations")
  --exclude               Exclude migrations by filename, comma separated
  --help                  Display help
  --json                  Format output as JSON
  --lock-mode             Set a lock mode [none, advisory-session] (default: "none")
  --no-versioning         Do not store version info in the database, just run the migrations
  --table                 Table name to store version info (default: "goose_db_version")
  --v                     Turn on verbose mode
  --version               Display the version of goose currently installed

ENVIRONMENT VARIABLES
  GOOSE_DBSTRING          Database connection string, lower priority than --dbstring
  GOOSE_DIR               Directory with migration files, lower priority than --dir

EXAMPLES
  goose --dbstring="postgres://dbuser:password1@localhost:5433/testdb?sslmode=disable" status
  goose --dbstring="mysql://user:password@/dbname?parseTime=true" status

  GOOSE_DIR=./examples/sql-migrations GOOSE_DBSTRING="sqlite:./test.db" goose status
  GOOSE_DBSTRING="clickhouse://user:password@localhost:9000/clickdb" goose status

LEARN MORE
  Use 'goose <command> --help' for more information about a command.
  Read the manual at https://pressly.github.io/goose/
`
)
