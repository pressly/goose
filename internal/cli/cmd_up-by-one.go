package cli

import (
	"context"
	"flag"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pressly/goose/v4"
)

func newUpByOneCmd(root *rootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("goose up-by-one", flag.ExitOnError)
	root.registerFlags(fs)
	pf := &providerFlags{}
	registerProviderFlags(fs, pf)

	usageOpt := &usageOpt{
		examples: []string{
			`$ goose up-by-one --dbstring="postgres://dbuser:password1@localhost:5433/testdb?sslmode=disable"`,
			`$ GOOSE_DIR=./examples/sql-migrations GOOSE_DBSTRING="sqlite:./test.db" goose up-by-one`,
		},
	}
	return &ffcli.Command{
		Name:       "up-by-one",
		ShortUsage: "goose up-by-one [flags]",
		ShortHelp:  "Migrate database up by one version",
		LongHelp:   upByOneLongHelp,
		FlagSet:    fs,
		UsageFunc:  newUsageFunc(usageOpt),
		Exec:       execUpByOneCmd(root, pf),
	}
}

func execUpByOneCmd(root *rootConfig, pf *providerFlags) func(context.Context, []string) error {
	return func(ctx context.Context, args []string) error {
		provider, err := newGooseProvider(root, pf)
		if err != nil {
			return err
		}
		now := time.Now()
		result, err := provider.UpByOne(ctx)
		if err != nil {
			return err
		}
		return printResult([]*goose.MigrationResult{result}, err, time.Since(now), root.useJSON)
	}
}

const upByOneLongHelp = `
Apply the next available migration.
`
