package goosecli

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/mfridman/buildversion"
	"github.com/mfridman/cli"
)

var root = &cli.Command{
	UsageFunc: rootUsageFunc(),

	Name:      "goose",
	ShortHelp: "A database migration tool for SQL databases.",
	Usage:     "goose <command> [flags] [args...]",
	Flags: cli.FlagsFunc(func(f *flag.FlagSet) {
		f.Bool("json", false, "Output in json format")
		f.Bool("version", false, "Print goose version and exit")
	}),
	Exec: func(ctx context.Context, s *cli.State) error {
		if cli.GetFlag[bool](s, "version") {
			fmt.Fprintf(s.Stdout, "goose version: %s\n", buildversion.New())
			return nil
		}
		if len(s.Args) == 0 {
			return errors.New("must supply a command to goose, see --help for more information")
		}
		return nil
	},
}

func rootUsageFunc() func(c *cli.Command) string {
	return func(c *cli.Command) string {
		return newHelp().
			add("", shortHelpSection).
			add("USAGE", usageSection).
			add("COMMANDS", commandsSection).
			add("GLOBAL FLAGS", flagsSection).
			add("SUPPORTED DATABASES", databasesSection).
			add("ENVIRONMENT VARIABLES (flags take precedence)", envVarsSection).
			add("LEARN MORE", learnMoreSection).
			build(c)
	}
}
