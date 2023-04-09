package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func newEnvCmd(root *rootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("goose env", flag.ExitOnError)
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:    "env",
		FlagSet: fs,
		Exec:    execEnvCmd(root),
	}
}

func execEnvCmd(root *rootConfig) func(ctx context.Context, args []string) error {
	return func(ctx context.Context, args []string) error {
		for _, env := range List() {
			fmt.Printf("%s=%q\n", env.Name, env.Value)
		}
		return nil
	}
}
