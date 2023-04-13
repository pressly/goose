package cli

import (
	"context"
	"flag"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func newUpCmd(root *rootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("goose up", flag.ExitOnError)
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:       "up",
		ShortUsage: "goose up [flags]",
		FlagSet:    fs,
		LongHelp:   "The up command runs all available migrations.",
		UsageFunc: defaultUsageFunc([]string{
			`$ goose up --dbstring="postgres://user:password@localhost:5432/dbname?sslmode=disable" --dir=db/migrations`,
			`$ GOOSE_DIR=./examples/sql-migrations GOOSE_DBSTRING="sqlite:./test.db" goose up`,
		}),
		Exec: execUpCmd(root),
	}
}

func execUpCmd(root *rootConfig) func(context.Context, []string) error {
	return func(ctx context.Context, args []string) error {
		provider, err := newGooseProvider(root)
		if err != nil {
			return err
		}
		now := time.Now()
		results, err := provider.Up(ctx)
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
