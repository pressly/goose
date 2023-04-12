package cli

import (
	"context"
	"flag"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pressly/goose/v4"
)

func newUpByOneCmd(root *rootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("goose up-by-one", flag.ExitOnError)
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:      "up-by-one",
		FlagSet:   fs,
		UsageFunc: func(c *ffcli.Command) string { return upByOneCmdUsage },
		Exec:      execUpByOneCmd(root),
	}
}

func execUpByOneCmd(root *rootConfig) func(context.Context, []string) error {
	return func(ctx context.Context, args []string) error {
		provider, err := newGooseProvider(root)
		if err != nil {
			return err
		}
		now := time.Now()
		result, err := provider.UpByOne(ctx)
		if err != nil {
			return err
		}
		return printMigrationResult(
			[]*goose.MigrationResult{result},
			time.Since(now),
			root.useJSON,
		)
	}
}

const (
	upByOneCmdUsage = `
Apply the next available migration.

USAGE
  goose up-by-one [flags]

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

EXAMPLES
  $ goose up-by-one --dbstring="postgres://dbuser:password1@localhost:5433/testdb?sslmode=disable"
  $ GOOSE_DIR=./examples/sql-migrations GOOSE_DBSTRING="sqlite:./test.db" goose up-by-one
`
)
