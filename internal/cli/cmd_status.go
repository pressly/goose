package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type statusCmd struct {
	root *rootConfig
}

func newStatusCmd(root *rootConfig) *ffcli.Command {
	c := statusCmd{root: root}
	fs := flag.NewFlagSet("goose status", flag.ExitOnError)
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:       "status",
		ShortUsage: "goose [flags] status",
		LongHelp:   "",
		FlagSet:    fs,
		Options: []ff.Option{
			ff.WithEnvVarPrefix("GOOSE"),
		},

		Exec: c.Exec,
	}
}

func (c *statusCmd) Exec(ctx context.Context, args []string) error {
	provider, err := newGooseProvider(c.root)
	if err != nil {
		return err
	}
	results, err := provider.Status(ctx, nil)
	if err != nil {
		return err
	}
	for _, result := range results {
		fmt.Println(result)
	}
	return nil
}

type statusesOutput struct {
	Statuses      []statusOutput `json:"statuses"`
	TotalDuration int64          `json:"total_duration_ms"`
}

type statusOutput struct {
	Type      string `json:"migration_type"`
	Version   int64  `json:"version"`
	AppliedAt string `json:"applied_at"`
	Filename  string `json:"filename"`
}
