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
	pf := &providerFlags{}
	registerProviderFlags(fs, pf)

	usageOpt := &usageOpt{
		examples: []string{
			`$ goose redo --dbstring="sqlite:./test.db" -dir=./examples/sql-migrations`,
		},
	}
	return &ffcli.Command{
		Name:       "redo",
		ShortUsage: "goose redo [flags]",
		ShortHelp:  "Roll back the last appied migration and re-apply it",
		LongHelp:   redoCmdLongHelp,
		FlagSet:    fs,
		UsageFunc:  newUsageFunc(usageOpt),
		Exec:       execRedoCmd(root, pf),
	}
}

func execRedoCmd(root *rootConfig, pf *providerFlags) func(context.Context, []string) error {
	return func(ctx context.Context, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("too many arguments")
		}
		provider, err := newGooseProvider(root, pf)
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

const redoCmdLongHelp = `
Rerun the most recently applied migration.

This is effectively "goose down" followed by "goose up-by-one".
`
