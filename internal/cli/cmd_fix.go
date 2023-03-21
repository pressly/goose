package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pressly/goose/v4"
)

func newFixCmd(root *rootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("goose fix", flag.ExitOnError)
	root.registerFlags(fs)
	var dir string
	fs.StringVar(&dir, "dir", "", "directory with migration files")

	usageOpt := &usageOpt{
		examples: []string{
			`$ goose fix --dir=./schema/migrations`,
			`$ GOOSE_DIR=./schema/migrations goose fix`,
		},
	}
	return &ffcli.Command{
		Name:       "fix",
		ShortUsage: "goose fix [flags]",
		ShortHelp:  "Apply sequential numbering to existing timestamped migrations",
		LongHelp:   fixCmdLongHelp,
		FlagSet:    fs,
		UsageFunc:  newUsageFunc(usageOpt),
		Exec:       execFixCmd(root, dir),
	}
}

func execFixCmd(root *rootConfig, dir string) func(ctx context.Context, args []string) error {
	return func(ctx context.Context, args []string) error {
		dir = coalesce(dir, GOOSE_DIR)
		if dir == "" {
			return fmt.Errorf("goose fix requires a migrations directory: %w", errNoDir)
		}
		fixResults, err := goose.Fix(dir)
		if err != nil {
			return err
		}
		for _, f := range fixResults {
			fmt.Fprintln(os.Stdout, "renamed", f.OldPath)
			fmt.Fprintln(os.Stdout, "    ==>", f.NewPath)
		}
		return nil
	}
}

const fixCmdLongHelp = `
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
`
