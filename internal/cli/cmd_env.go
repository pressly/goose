package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type envCmd struct {
	root *rootConfig
}

func newEnvCmd(root *rootConfig) *ffcli.Command {
	c := envCmd{root: root}
	fs := flag.NewFlagSet("goose env", flag.ExitOnError)
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:       "env",
		ShortUsage: "goose env",
		LongHelp:   "",
		ShortHelp:  "",
		Exec:       c.Exec,
		FlagSet:    fs,
		Options: []ff.Option{
			ff.WithEnvVarPrefix("GOOSE"),
		},
	}
}

func (c *envCmd) Exec(ctx context.Context, args []string) error {
	for _, env := range List() {
		fmt.Printf("%s=%q\n", env.Name, env.Value)
	}
	return nil
}
