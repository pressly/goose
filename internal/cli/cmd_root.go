package cli

import (
	"context"
	"flag"
	"io"

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

	// stdout is the output stream for the command. It is set to os.Stdout by
	// default, but can be overridden for testing.
	stdout io.Writer
}

func newRootCmd(w io.Writer) (*ffcli.Command, *rootConfig) {
	config := &rootConfig{
		stdout: w,
	}
	fs := flag.NewFlagSet("goose", flag.ExitOnError)
	config.registerFlags(fs)

	root := &ffcli.Command{
		Name:    "goose [flags] <subcommand>",
		FlagSet: fs,
		Options: []ff.Option{
			ff.WithEnvVarPrefix("GOOSE"),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
	return root, config
}

// registerFlags registers the flag fields into the provided flag.FlagSet. This
// helper function allows subcommands to register the root flags into their
// flagsets, creating "global" flags that can be passed after any subcommand at
// the commandline.
func (c *rootConfig) registerFlags(fs *flag.FlagSet) {
	fs.BoolVar(&c.verbose, "v", false, "log verbose output")
	fs.BoolVar(&c.useJSON, "json", false, "log output as JSON")
	// Migration configuration
	fs.StringVar(&c.dir, "dir", DefaultDir, "directory with migration files")
	// Database configuration
	fs.StringVar(&c.dbstring, "dbstring", "", "database connection string")
	fs.StringVar(&c.table, "table", "goose_db_version", "database table to store version info")
	// Features
	fs.BoolVar(&c.noVersioning, "no-versioning", false, "do not use versioning")
	fs.BoolVar(&c.allowMissing, "allow-missing", false, "allow missing (out-of-order) migrations")
	fs.StringVar(&c.lockMode, "lock-mode", "none", "lock mode (none, advisory-session)")
	fs.Var(&c.excludeFilenames, "exclude", "exclude filenames (comma separated)")

	// fs.BoolVar(&c.grouped, "grouped", false, "run migrations in transaction groups, if possible")
}
