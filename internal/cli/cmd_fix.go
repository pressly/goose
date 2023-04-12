package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pressly/goose/v4"
)

func newFixCmd(root *rootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("goose fix", flag.ExitOnError)
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:      "fix",
		FlagSet:   fs,
		UsageFunc: func(c *ffcli.Command) string { return fixCmdUsage },
		Exec:      execFixCmd(root),
	}
}

func execFixCmd(root *rootConfig) func(ctx context.Context, args []string) error {
	return func(ctx context.Context, args []string) error {
		fixResults, err := goose.Fix(root.dir)
		if err != nil {
			return err
		}
		for _, f := range fixResults {
			fmt.Println("renamed", f.OldPath)
			fmt.Println("    ==>", f.NewPath)
		}

		// TODO(mf): add json output

		return nil
	}
}

const (
	fixCmdUsage = `
Rename all migration files to a sequential version number. The next migration version number is 
determined by the highest version number on disk.

Hybrid Versioning
=================
We strongly recommend adopting a hybrid versioning approach, using both timestamps and sequential 
numbers. Migrations created during the development process are timestamped and sequential versions 
are ran on production. We believe this method will prevent the problem of conflicting versions when 
writing software in a team environment.

To help you adopt this approach, goose create will use the current timestamp as the migration 
version. When you're ready to deploy migrations in a production environment, we also provide a 
helpful fix command to convert migrations into sequential order, while preserving the timestamp 
ordering. We recommend running goose fix in the CI pipeline, and only when the migrations are ready 
for production.

USAGE:
  goose fix [flags]

FLAGS:
  --dir       Directory with migration files (default: "./migrations")
  --json      Output results as JSON
  --v         Turn on verbose mode

Examples:
  $ goose fix --dir=./schema/migrations
  $ GOOSE_DIR=./schema/migrations goose fix
`
)
