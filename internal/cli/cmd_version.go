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
	registerFlags(fs, root)
	pf := &providerFlags{}
	registerProviderFlags(fs, pf)

	usageOpt := &usageOpt{
		examples: []string{
			`$ goose version --dbstring="postgres://user:password@localhost:5432/dbname" --dir=db/migrations`,
			`$ GOOSE_DIR=./examples/sql-migrations GOOSE_DBSTRING="sqlite:./test.db" goose version`,
		},
	}
	return &ffcli.Command{
		Name:       "version",
		ShortUsage: "goose version [flags]",
		ShortHelp:  "Print the current version of the database",
		LongHelp:   versionCmdLongHelp,
		FlagSet:    fs,
		UsageFunc:  newUsageFunc(usageOpt),
		Exec:       execVersionCmd(root, pf),
	}
}

func execVersionCmd(root *rootConfig, pf *providerFlags) func(context.Context, []string) error {
	return func(ctx context.Context, args []string) error {
		provider, err := newGooseProvider(root, pf)
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

const versionCmdLongHelp = `
Print the current version of the database.

Note, if using --allow-missing, this command will return the recently applied migration, rather than 
the highest applied migration by version.
`
