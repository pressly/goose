package cli

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func newRedoCmd(root *rootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("goose redo", flag.ExitOnError)
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:      "redo",
		FlagSet:   fs,
		UsageFunc: func(c *ffcli.Command) string { return redoCmdUsage },
		Exec:      execRedoCmd(root),
	}
}

func execRedoCmd(root *rootConfig) func(context.Context, []string) error {
	return func(ctx context.Context, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("too many arguments")
		}
		provider, err := newGooseProvider(root)
		if err != nil {
			return err
		}
		now := time.Now()
		results, err := provider.Redo(ctx)
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
	redoCmdUsage = `
Rerun the most recently applied migration.

USAGE
  goose redo [flags]

FLAGS
  --dbstring           Database connection string
  --dir                Directory with migration files (default: "./migrations")
  --exclude            Exclude migrations by filename, comma separated
  --json               Format output as JSON
  --lock-mode          Set a lock mode [none, advisory-session] (default: "none")
  --no-versioning      Do not store version info in the database, just run the migrations
  --table              Table name to store version info (default: "goose_db_version")
  --v                  Turn on verbose mode

EXAMPLES
  $ goose redo --dbstring="sqlite:./test.db"
`
)
