package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func newStatusCmd(root *rootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("goose status", flag.ExitOnError)
	root.registerFlags(fs)
	pf := &providerFlags{}
	registerProviderFlags(fs, pf)

	usageOpt := &usageOpt{
		examples: []string{
			`$ goose status --dir=./schema/migrations --dbstring="sqlite:./test.db"`,
			`$ GOOSE_DIR=./schema/migrations GOOSE_DBSTRING="sqlite:./test.db" goose status`,
		},
	}
	return &ffcli.Command{
		Name:       "status",
		ShortUsage: "goose status [flags]",
		ShortHelp:  "List applied and pending migrations",
		LongHelp:   statusCmdLongHelp,
		FlagSet:    fs,
		UsageFunc:  newUsageFunc(usageOpt),
		Exec:       execStatusCmd(root, pf),
	}
}

func execStatusCmd(root *rootConfig, pf *providerFlags) func(ctx context.Context, args []string) error {
	return func(ctx context.Context, args []string) error {
		provider, err := newGooseProvider(root, pf)
		if err != nil {
			return err
		}
		results, err := provider.Status(ctx, nil)
		if err != nil {
			return err
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
		defer tw.Flush()
		fmtPattern := "%v\t%v\n"
		fmt.Fprintf(tw, fmtPattern, "Applied At", "Migration")
		fmt.Fprintf(tw, fmtPattern, "──────────", "─────────")
		for _, result := range results {
			t := "Pending"
			if result.Applied {
				t = result.AppliedAt.Format("2006-01-02 15:04:05")
			}
			fmt.Fprintf(tw, fmtPattern,
				t,
				filepath.Base(result.Source.Fullpath),
			)
		}
		return nil
	}
}

//lint:ignore U1000 Ignore unused code for now
type statusesOutput struct {
	Statuses      []statusOutput `json:"statuses"`
	TotalDuration int64          `json:"total_duration_ms"`
}

//lint:ignore U1000 Ignore unused code for now
type statusOutput struct {
	Type      string `json:"migration_type"`
	Version   int64  `json:"version"`
	AppliedAt string `json:"applied_at"`
	Filename  string `json:"filename"`
}

const statusCmdLongHelp = `
List the status of all migrations, comparing the current state of the database with the migrations 
on disk. 

If a migration is on disk but is not applied to the database, it will be listed as "Pending".

Note, if --allow-missing is set, this command will report migrations as "out-of-order". This
surfaces migration versions that are lower than the current database version, but are not applied
to the database.
`
