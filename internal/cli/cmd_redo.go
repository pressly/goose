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

	return &ffcli.Command{
		Name:    "redo",
		FlagSet: fs,
		Exec:    execRedoCmd(root),
	}
}

func execRedoCmd(root *rootConfig) func(context.Context, []string) error {
	return func(ctx context.Context, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("too many arguments")
		}
		provider, err := newGooseProvider(root)
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
