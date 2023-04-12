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
		Name:      "env",
		FlagSet:   fs,
		UsageFunc: func(c *ffcli.Command) string { return envCmdUsage },
		Exec:      execEnvCmd(root),
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

const (
	envCmdUsage = `
Print the environment variables used by goose and their values.

USAGE
  goose env

ENVIRONMENT VARIABLES
  GOOSE_DBSTRING          Database connection string, lower priority than --dbstring
  GOOSE_DIR               Directory with migration files, lower priority than --dir
  NO_COLOR                Disable color output, lower priority than --no-color
`
)
