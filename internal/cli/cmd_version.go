package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type versionCmd struct {
	root *rootConfig
}

func newVersionCmd(root *rootConfig) *ffcli.Command {
	c := versionCmd{root: root}
	fs := flag.NewFlagSet("goose version", flag.ExitOnError)
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:       "version",
		ShortUsage: "goose [flags] version",
		LongHelp:   "",
		FlagSet:    fs,
		Options: []ff.Option{
			ff.WithEnvVarPrefix("GOOSE"),
		},

		Exec: c.Exec,
	}
}

func (c *versionCmd) Exec(ctx context.Context, args []string) error {
	provider, err := newGooseProvider(c.root)
	if err != nil {
		return err
	}
	now := time.Now()
	version, err := provider.GetDBVersion(ctx)
	if err != nil {
		return err
	}

	if c.root.useJSON {
		type versionOutput struct {
			Version       int64 `json:"version"`
			TotalDuration int64 `json:"total_duration_ms"`
		}
		data := versionOutput{
			Version:       version,
			TotalDuration: time.Since(now).Milliseconds(),
		}
		return json.NewEncoder(os.Stdout).Encode(data)
	} else {
		fmt.Println("goose: version ", version)
	}
	return nil
}
