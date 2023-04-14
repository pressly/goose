package cli

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func newDownToCmd(root *rootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("goose down-to", flag.ExitOnError)
	registerFlags(fs, root)
	pf := &providerFlags{}
	registerProviderFlags(fs, pf)

	usageOpt := &usageOpt{
		examples: []string{
			`$ GOOSE_DBSTRING="postgres://localhost:5432/mydb" goose down-to 0`,
			`$ goose down-to --dir=./examples/sql-migrations --json --dbstring="sqlite:./test.db" 0`,
		},
	}
	return &ffcli.Command{
		Name:       "down-to",
		ShortUsage: "goose down-to [flags] <version>",
		ShortHelp:  "Migrate database down to, but not including, a specific version",
		LongHelp:   downToCmdLongHelp,
		FlagSet:    fs,
		UsageFunc:  newUsageFunc(usageOpt),
		Exec:       execDownToCmd(root, pf),
	}
}

func execDownToCmd(root *rootConfig, pf *providerFlags) func(ctx context.Context, args []string) error {
	return func(ctx context.Context, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("missing required argument: version")
		}
		version, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid version: %s, must be an integer", args[0])
		}
		provider, err := newGooseProvider(root, pf)
		if err != nil {
			return err
		}
		now := time.Now()
		results, err := provider.DownTo(ctx, version)
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

const downToCmdLongHelp = `
The command goose down-to 0 will apply all down migrations.

Note, when applying missing (out-of-order) up migrations, goose down-to will migrate them back down 
in the order they were originally applied, and not by the version order. For example, if applied
migrations 1,3,2,4 then goose down-to 0 will apply migrations 4,2,3,1.
`
