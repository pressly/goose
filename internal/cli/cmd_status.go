package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func newStatusCmd(root *rootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("goose status", flag.ExitOnError)
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:      "status",
		FlagSet:   fs,
		UsageFunc: func(c *ffcli.Command) string { return statusCmdUsage },
		Exec:      execStatusCmd(root),
	}
}

func execStatusCmd(root *rootConfig) func(ctx context.Context, args []string) error {
	return func(ctx context.Context, args []string) error {
		provider, err := newGooseProvider(root)
		if err != nil {
			return err
		}
		_ = statusesOutput{}
		results, err := provider.Status(ctx, nil)
		if err != nil {
			return err
		}
		for _, result := range results {
			fmt.Println(result)
		}
		return nil
	}
}

type statusesOutput struct {
	Statuses      []statusOutput `json:"statuses"`
	TotalDuration int64          `json:"total_duration_ms"`
}

type statusOutput struct {
	Type      string `json:"migration_type"`
	Version   int64  `json:"version"`
	AppliedAt string `json:"applied_at"`
	Filename  string `json:"filename"`
}

const statusCmdUsage = `
List the status of all migrations, comparing the current state of the database with the migrations 
on disk. 

If a migration is on disk but is not applied to the database, it will be listed as "pending".

Note, if --allow-missing is set, this command will report migrations as "out-of-order". This
surfaces migration versions that are lower than the current database version, but are not applied
to the database.


USAGE
  goose status [flags]

FLAGS
  --allow-missing      Allow missing (out-of-order) migrations
  --dbstring           Database connection string
  --dir                Directory with migration files (default: "./migrations")
  --exclude            Exclude migrations by filename, comma separated
  --json               Format output as JSON
  --lock-mode          Set a lock mode [none, advisory-session] (default: "none")
  --table              Table name to store version info (default: "goose_db_version")
  --v                  Turn on verbose mode

EXAMPLES
  $ goose status --dir=./schema/migrations --dbstring="sqlite:./test.db"
  $ GOOSE_DIR=./schema/migrations GOOSE_DBSTRING="sqlite:./test.db" goose status
`
