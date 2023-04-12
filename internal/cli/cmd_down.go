package cli

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pressly/goose/v4"
)

func newDownCmd(root *rootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("goose down", flag.ExitOnError)
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:      "down",
		FlagSet:   fs,
		UsageFunc: func(c *ffcli.Command) string { return downCmdUsage },
		Exec:      execDownCmd(root),
	}
}

func execDownCmd(root *rootConfig) func(context.Context, []string) error {
	return func(ctx context.Context, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("too many arguments")
		}
		provider, err := newGooseProvider(root)
		if err != nil {
			return err
		}
		now := time.Now()
		result, err := provider.Down(ctx)
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
	downCmdUsage = `
Migrate the database down by one version.

Note, when applying missing (out-of-order) up migrations, goose down will migrate them back down in 
the order they were originally applied, and not by the version order. For example, if applied 
migrations 1,3,2,4 then goose down-to 0 will apply migrations 4,2,3,1.

USAGE
  goose down [flags]

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
  $ GOOSE_DBSTRING="postgres://localhost:5432/mydb" goose down
  $ goose down --dir=./examples/sql-migrations --json --dbstring="sqlite:./test.db"
`
)
