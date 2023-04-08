package cli

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type redoCmd struct {
	root *rootConfig
}

func newRedoCmd(root *rootConfig) *ffcli.Command {
	c := redoCmd{root: root}
	fs := flag.NewFlagSet("goose redo", flag.ExitOnError)
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:       "redo",
		ShortUsage: "goose [flags] redo",
		LongHelp:   "",
		FlagSet:    fs,
		Options: []ff.Option{
			ff.WithEnvVarPrefix("GOOSE"),
		},

		Exec: c.Exec,
	}
}

func (c *redoCmd) Exec(ctx context.Context, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments")
	}
	provider, err := newGooseProvider(c.root)
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
		c.root.useJSON,
	)
}
