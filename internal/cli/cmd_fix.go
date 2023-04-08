package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pressly/goose/v4"
)

type fixCmd struct {
	root *rootConfig
}

func newFixCmd(root *rootConfig) *ffcli.Command {
	c := fixCmd{root: root}
	fs := flag.NewFlagSet("goose fix", flag.ExitOnError)
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:       "fix",
		ShortUsage: "goose [flags] fix",
		LongHelp:   "",
		ShortHelp:  "",
		Exec:       c.Exec,
		FlagSet:    fs,
		Options: []ff.Option{
			ff.WithEnvVarPrefix("GOOSE"),
		},
	}
}

func (c *fixCmd) Exec(ctx context.Context, args []string) error {
	fixResults, err := goose.Fix(c.root.dir)
	if err != nil {
		return err
	}
	for _, f := range fixResults {
		fmt.Println("renamed", f.OldPath)
		fmt.Println("    ==>", f.NewPath)
	}

	// TODO(mf): add json output

	return nil
}
