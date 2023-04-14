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
	registerFlags(fs, root)
	pf := &providerFlags{}
	// TODO: not all provider flags apply
	registerProviderFlags(fs, pf)

	usageOpt := &usageOpt{
		examples: []string{
			`$ GOOSE_DBSTRING="postgres://localhost:5432/mydb" goose down`,
			`$ goose down --dir=./examples/sql-migrations --json --dbstring="sqlite:./test.db"`,
		},
	}
	return &ffcli.Command{
		Name:       "down",
		ShortUsage: "goose down [flags]",
		ShortHelp:  "Migrate the database down by one version",
		LongHelp:   downCmdLongHelp,
		FlagSet:    fs,
		UsageFunc:  newUsageFunc(usageOpt),
		Exec:       execDownCmd(root, pf),
	}
}

func execDownCmd(root *rootConfig, pf *providerFlags) func(context.Context, []string) error {
	return func(ctx context.Context, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("too many arguments")
		}
		provider, err := newGooseProvider(root, pf)
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

const downCmdLongHelp = `
Migrate the database down by one version.

Note, when applying missing (out-of-order) up migrations, goose down will migrate them back down in 
the order they were originally applied, and not by the version order. For example, if applied 
migrations 1,3,2,4 then goose down-to 0 will apply migrations 4,2,3,1.
`
