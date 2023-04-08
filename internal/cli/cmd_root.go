package cli

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type rootConfig struct {
	dir     string
	verbose bool
	useJSON bool

	dbstring         string
	table            string
	noVersioning     bool
	allowMissing     bool
	lockMode         string
	grouped          bool
	excludeFilenames stringSet
}

func newRootCmd() (*ffcli.Command, *rootConfig) {
	config := new(rootConfig)
	fs := flag.NewFlagSet("goose", flag.ExitOnError)
	config.registerFlags(fs)

	root := &ffcli.Command{
		Name:    "goose [flags] <subcommand>",
		FlagSet: fs,
		Exec:    config.Exec,
		Options: []ff.Option{
			ff.WithEnvVarPrefix("GOOSE"),
		},
	}
	return root, config
}

// registerFlags registers the flag fields into the provided flag.FlagSet. This
// helper function allows subcommands to register the root flags into their
// flagsets, creating "global" flags that can be passed after any subcommand at
// the commandline.
func (c *rootConfig) registerFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.dir, "dir", DefaultDir, "directory with migration files")
	fs.BoolVar(&c.verbose, "v", false, "log verbose output")
	fs.BoolVar(&c.useJSON, "json", false, "log output as JSON")

	fs.StringVar(&c.dbstring, "dbstring", "", "database connection string")
	fs.StringVar(&c.table, "table", "goose_db_version", "database table to store version info")
	fs.BoolVar(&c.noVersioning, "no-versioning", false, "do not use versioning")
	fs.StringVar(&c.lockMode, "lock-mode", "none", "lock mode (none, advisory-session)")
	fs.BoolVar(&c.grouped, "grouped", false, "run migrations in transaction groups, if possible")
	fs.Var(&c.excludeFilenames, "exclude", "exclude filenames (comma separated)")
	fs.BoolVar(&c.allowMissing, "allow-missing", false, "allow missing (out-of-order) migrations")
}

func (c *rootConfig) Exec(context.Context, []string) error {
	return flag.ErrHelp
}
