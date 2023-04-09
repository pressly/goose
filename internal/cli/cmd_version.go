package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func newVersionCmd(root *rootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("goose version", flag.ExitOnError)
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:    "version",
		FlagSet: fs,
		Exec:    execVersionCmd(root),
	}
}

func execVersionCmd(root *rootConfig) func(context.Context, []string) error {
	return func(ctx context.Context, args []string) error {
		provider, err := newGooseProvider(root)
		if err != nil {
			return err
		}
		now := time.Now()
		version, err := provider.GetDBVersion(ctx)
		if err != nil {
			return err
		}
		if root.useJSON {
			data := versionOutput{
				Version:       version,
				TotalDuration: time.Since(now).Milliseconds(),
			}
			return json.NewEncoder(os.Stdout).Encode(data)
		}
		fmt.Println("goose: version ", version)
		return nil
	}
}

type versionOutput struct {
	Version       int64 `json:"version"`
	TotalDuration int64 `json:"total_duration_ms"`
}
