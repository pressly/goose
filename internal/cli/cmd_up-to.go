package cli

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func newUpToCmd(root *rootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("goose up-to", flag.ExitOnError)
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:      "up-to",
		FlagSet:   fs,
		UsageFunc: func(c *ffcli.Command) string { return upToCmdUsage },
		Exec:      execUpToCmd(root),
	}
}

func execUpToCmd(root *rootConfig) func(ctx context.Context, args []string) error {
	return func(ctx context.Context, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("missing required argument: version")
		}
		version, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid version: %s, must be an integer", args[0])
		}
		provider, err := newGooseProvider(root)
		if err != nil {
			return err
		}
		now := time.Now()
		results, err := provider.UpTo(ctx, version)
		if err != nil {
			return err
		}
		return printMigrationResult(
			results,
			time.Since(now),
			root.useJSON,
		)
	}
}

const (
	upToCmdUsage = `
Apply available migrations up to, and including, the specified version.

USAGE
  goose up-to [flags] <version>

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
  $ goose up-to --dbstring="sqlite:./test.db" 42
  $ GOOSE_DIR=./examples/sql-migrations GOOSE_DBSTRING="sqlite:./test.db" goose up-to 3
`
)
