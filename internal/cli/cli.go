package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pressly/goose/v4"
)

var commands = map[string]func(*rootConfig) *ffcli.Command{
	"create":    newCreateCmd,
	"down":      newDownCmd,
	"down-to":   newDownToCmd,
	"env":       newEnvCmd,
	"fix":       newFixCmd,
	"redo":      newRedoCmd,
	"status":    newStatusCmd,
	"up-by-one": newUpByOneCmd,
	"up-to":     newUpToCmd,
	"up":        newUpCmd,
	"version":   newVersionCmd,
}

// Run is the entry point for the goose CLI. It parses the command line arguments and executes the
// appropriate command.
//
// The supplied args should not include the name of the executable.
func Run(args []string) error {
	rootCmd, rootConfig := newRootCmd(os.Stdout)
	// Add subcommands. These will be advertised in the help text.
	for _, cmd := range commands {
		rootCmd.Subcommands = append(rootCmd.Subcommands, cmd(rootConfig))
	}
	// Set the usage function for all subcommands, if not already set.
	for _, c := range rootCmd.Subcommands {
		if c.UsageFunc == nil {
			c.UsageFunc = usageFunc
		}
	}
	// Parse the command line flags.
	if err := rootCmd.Parse(args); err != nil {
		return fmt.Errorf("parsing error: %w", err)
	}

	// This is where we can validate the root config after parsing the command line flags.
	_ = rootConfig

	// Here we can add additional commands but not advertised in the help text.

	return rootCmd.Run(context.Background())
}

type stringSet []string

func (ss *stringSet) Set(value string) error {
	for _, existing := range *ss {
		if value == existing {
			return errors.New("duplicate")
		}
	}
	(*ss) = append(*ss, value)
	return nil
}

func (ss *stringSet) String() string {
	return strings.Join(*ss, ", ")
}

// func truncateDuration(d time.Duration) time.Duration {
//  for _, v := range []time.Duration{
//      time.Second,
//      time.Millisecond,
//      time.Microsecond,
//  } {
//      if d > v {
//          return d.Round(v / time.Duration(100))
//      }
//  }
//  return d
// }

func newGooseProvider(root *rootConfig) (*goose.Provider, error) {
	db, gooseDialect, err := openConnection(root.dbstring)
	if err != nil {
		return nil, err
	}
	opt := goose.DefaultOptions().
		SetVerbose(root.verbose).
		SetNoVersioning(root.noVersioning).
		SetAllowMissing(root.allowMissing)

	if len(root.excludeFilenames) > 0 {
		opt = opt.SetExcludeFilenames(root.excludeFilenames...)
	}
	if root.dir != "" {
		opt = opt.SetDir(root.dir)
	}
	if root.table != "" {
		opt = opt.SetTableName(root.table)
	}
	if root.lockMode != "" {
		var lockMode goose.LockMode
		switch root.lockMode {
		case "none":
			lockMode = goose.LockModeNone
		case "advisory-session":
			lockMode = goose.LockModeAdvisorySession
		default:
			return nil, fmt.Errorf("invalid lock mode: %s", root.lockMode)
		}
		opt = opt.SetLockMode(lockMode)
	}

	return goose.NewProvider(gooseDialect, db, opt)
}

func usageFunc(c *ffcli.Command) string {
	return usage
}

const (
	usage = `goose

A database migration tool that simplifies the process of versioning, applying, and rolling back
schema changes in a controlled and organized way.

USAGE
  goose [flags] <command>

COMMANDS
  create          Create a new migration file
  down            Migrate down previous version
  down-to         Migrate database down to, but not including, a specific version
  env             Print environment variables
  fix             Apply sequential numbering to existing timestamped migrations
  redo            Roll back the last migration and re-apply it
  status          List applied and unapplied migrations
  up-by-one       Migrate database up by one version
  up-to           Migrate database up to, and including, a specific version
  up              Migrate database to the most recent version
  version         Print the current version of the database

SUPPORTED DATABASES
  postgres       mysql       sqlite3
  redshift       tidb        mssql
  clickhouse     vertica

FLAGS
  --allow-missing         Allow missing (out-of-order) migrations
  --dbstring              Database connection string
  --dir                   Directory with migration files (default: "./migrations")
  --exclude               Exclude migrations by filename, comma separated
  --help                  Show help for command
  --json                  Output formatted JSON
  --lock-mode             Set a lock mode [none, advisory-session] (default: "none")
  --no-versioning         Do not store version info in the database, just run the migrations
  --table                 Table name to store version info (default: "goose_db_version")
  --v                     Turn on verbose mode
  --version               Display the version of goose currently installed

ENVIRONMENT VARIABLES
  GOOSE_DBSTRING          Database connection string, lower priority than --dbstring
  GOOSE_DIR               Directory with migration files, lower priority than --dir

EXAMPLES
  goose --dbstring="postgres://user:password@localhost:5432/dbname?sslmode=disable" status
  goose --dbstring="mysql://user:password@/dbname?parseTime=true" status

  GOOSE_DIR=./examples/sql-migrations GOOSE_DBSTRING="sqlite:./test.db" goose status
  GOOSE_DBSTRING="clickhouse://tcp://127.0.0.1:900" goose status

LEARN MORE
  Use 'goose <command> --help' for more information about a command.
  Read the manual at https://pressly.github.io/goose/
`
)
