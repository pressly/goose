package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func newVersionCmd(root *rootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("goose version", flag.ExitOnError)
	root.registerFlags(fs)
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
		version, err := provider.GetDBVersion(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("goose: version %d\n", version)
		return nil
	}
}

//lint:ignore U1000 Ignore unused code for now
type versionOutput struct {
	Version       int64 `json:"version"`
	TotalDuration int64 `json:"total_duration_ms"`
}

const versionCmdLongHelp = `
Print the current version of the database.

Note, if using --allow-missing, this command will return the recently applied migration, rather than 
the highest applied migration by version.
`
