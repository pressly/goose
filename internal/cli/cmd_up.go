package cli

import (
	"context"
	"flag"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
)

type providerFlags struct {
	dir              string
	dbstring         string
	table            string
	noVersioning     bool
	allowMissing     bool
	lockMode         string
	excludeFilenames stringSet
}

func registerProviderFlags(fs *flag.FlagSet, p *providerFlags) {
	fs.StringVar(&p.dir, "dir", "", "")
	fs.StringVar(&p.dbstring, "dbstring", "", "")
	fs.StringVar(&p.table, "table", "", "")
	fs.BoolVar(&p.noVersioning, "no-versioning", false, "")
	fs.BoolVar(&p.allowMissing, "allow-missing", false, "")
	fs.StringVar(&p.lockMode, "lock-mode", "", "")
	fs.Var(&p.excludeFilenames, "exclude", "")
}

func newUpCmd(root *rootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("goose up", flag.ExitOnError)
	pf := new(providerFlags)
	registerProviderFlags(fs, pf)

	usageOpt := &usageOpt{
		examples: []string{
			`$ goose up --dbstring="postgres://user:password@localhost:5432/dbname?sslmode=disable" --dir=db/migrations`,
			`$ GOOSE_DIR=./examples/sql-migrations GOOSE_DBSTRING="sqlite:./test.db" goose up`,
		},
	}
	return &ffcli.Command{
		Name:       "up",
		ShortUsage: "goose up [flags]",
		ShortHelp:  "Migrate database to the most recent version",
		LongHelp:   upCmdLongHelp,
		FlagSet:    fs,
		UsageFunc:  newUsageFunc(usageOpt),
		Exec:       execUpCmd(root, pf),
	}
}

func execUpCmd(root *rootConfig, providerFlags *providerFlags) func(context.Context, []string) error {
	return func(ctx context.Context, args []string) error {
		provider, err := newGooseProvider(root, providerFlags)
		if err != nil {
			return err
		}
		now := time.Now()
		results, err := provider.Up(ctx)
		if err != nil {
			return err
		}
		return printMigrationResult(
			results,
			time.Since(now),
			root.useJSON,
		)
	}
}

const upCmdLongHelp = `
The up command runs all available migrations.
`
