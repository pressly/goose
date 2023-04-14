package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func newEnvCmd(root *rootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("goose env", flag.ExitOnError)

	usageOpt := &usageOpt{
		envs: []string{EnvGooseDBString, EnvGooseDir, EnvGooseTable, EnvNoColor},
	}
	return &ffcli.Command{
		Name:       "env",
		ShortUsage: "goose env",
		ShortHelp:  "Print environment variables used by goose and their values",
		LongHelp:   envCmdLongHelp,
		FlagSet:    fs,
		UsageFunc:  newUsageFunc(usageOpt),
		Exec: func(ctx context.Context, args []string) error {
			for _, env := range List() {
				fmt.Printf("%s=%q\n", env.Name, env.Value)
			}
			return nil
		},
	}
}

const envCmdLongHelp = `
Print the environment variables used by goose and their values.

If both a flag and an environment variable are set, the flag takes precedence.
`
