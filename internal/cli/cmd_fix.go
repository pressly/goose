package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pressly/goose/v4"
)

func newFixCmd(root *rootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("goose fix", flag.ExitOnError)
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:       "fix",
		ShortUsage: "goose [flags] fix",
		LongHelp:   "",
		ShortHelp:  "",
		FlagSet:    fs,
		Options: []ff.Option{
			ff.WithEnvVarPrefix("GOOSE"),
		},
		Exec: execFixCmd(root),
	}
}

func execFixCmd(root *rootConfig) func(ctx context.Context, args []string) error {
	return func(ctx context.Context, args []string) error {
		fixResults, err := goose.Fix(root.dir)
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
}
