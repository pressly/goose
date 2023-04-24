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
	pf := &providerFlags{}
	registerProviderFlags(fs, pf)

	usageOpt := &usageOpt{
		examples: []string{
			`$ goose up-to --dbstring="sqlite:./test.db" 42`,
			`$ GOOSE_DIR=./examples/sql-migrations GOOSE_DBSTRING="sqlite:./test.db" goose up-to 3`,
		},
	}
	return &ffcli.Command{
		Name:       "up-to",
		ShortUsage: "goose up-to [flags] <version>",
		ShortHelp:  "Migrate database up to, and including, a specific version",
		LongHelp:   upToLongHelp,
		FlagSet:    fs,
		UsageFunc:  newUsageFunc(usageOpt),
		Exec:       execUpToCmd(root, pf),
	}
}

func execUpToCmd(root *rootConfig, pf *providerFlags) func(ctx context.Context, args []string) error {
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
		results, err := provider.UpTo(ctx, version)
		return printResult(results, err, time.Since(now), root.useJSON)
	}
}

const upToLongHelp = `
Apply available migrations up to, and including, the specified version.
`
