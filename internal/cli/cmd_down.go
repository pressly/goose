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
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:    "down",
		FlagSet: fs,
		Exec:    execDownCmd(root),
	}
}

func execDownCmd(root *rootConfig) func(context.Context, []string) error {
	return func(ctx context.Context, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("too many arguments")
		}
		provider, err := newGooseProvider(root)
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
