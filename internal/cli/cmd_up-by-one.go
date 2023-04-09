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

	return &ffcli.Command{
		Name:    "up-by-one",
		FlagSet: fs,
		Exec:    execUpByOneCmd(root),
	}
}

func execUpByOneCmd(root *rootConfig) func(context.Context, []string) error {
	return func(ctx context.Context, args []string) error {
		provider, err := newGooseProvider(root)
		if err != nil {
			return err
		}
		now := time.Now()
		result, err := provider.UpByOne(ctx)
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
