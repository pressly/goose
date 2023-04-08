package cli

import (
	"context"
	"flag"
	"time"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pressly/goose/v4"
)

type upByOneCmd struct {
	root *rootConfig
}

func newUpByOneCmd(root *rootConfig) *ffcli.Command {
	c := upByOneCmd{root: root}
	fs := flag.NewFlagSet("goose up-by-one", flag.ExitOnError)
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:       "up-by-one",
		ShortUsage: "goose [flags] up-by-one",
		LongHelp:   "",
		ShortHelp:  "",
		Exec:       c.Exec,
		FlagSet:    fs,
		Options: []ff.Option{
			ff.WithEnvVarPrefix("GOOSE"),
		},
	}
}

func (c *upByOneCmd) Exec(ctx context.Context, args []string) error {
	provider, err := newGooseProvider(c.root)
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
		c.root.useJSON,
	)
}
