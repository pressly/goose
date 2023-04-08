package cli

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pressly/goose/v4"
)

type downCmd struct {
	root *rootConfig
}

func newDownCmd(root *rootConfig) *ffcli.Command {
	c := downCmd{root: root}
	fs := flag.NewFlagSet("goose down", flag.ExitOnError)
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:       "down",
		ShortUsage: "goose [flags] down",
		LongHelp:   "",
		FlagSet:    fs,
		Options: []ff.Option{
			ff.WithEnvVarPrefix("GOOSE"),
		},

		Exec: c.Exec,
	}
}

func (c *downCmd) Exec(ctx context.Context, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments")
	}
	provider, err := newGooseProvider(c.root)
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
		c.root.useJSON,
	)
}
