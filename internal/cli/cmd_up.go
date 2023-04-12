package cli

import (
	"context"
	"flag"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func newUpCmd(root *rootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("goose up", flag.ExitOnError)
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:      "up",
		FlagSet:   fs,
		UsageFunc: func(c *ffcli.Command) string { return upCmdUsage },
		Exec:      execUpCmd(root),
	}
}

func execUpCmd(root *rootConfig) func(context.Context, []string) error {
	return func(ctx context.Context, args []string) error {
		provider, err := newGooseProvider(root)
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

type cmdUsage struct {
	Short      string
	Additional string
	Usage      string
	Flags      []string
	Examples   []string
}

type flagUsage struct {
	Name     string
	Short    string
	Availble []string
	Default  string
}

var (
	upCmdUsageX = cmdUsage{
		Short:      "The up command runs all available migrations.",
		Additional: "",
		Usage:      "goose up [flags]",
	}
)

var flagLookup = map[string]flagUsage{
	"allow-missing": {
		Name:    "--allow-missing",
		Short:   "Allow missing (out-of-order) migrations",
		Default: "",
	},
	"dbstring": {
		Name:    "--dbstring",
		Short:   "Database connection string",
		Default: "",
	},
	"dir": {
		Name:    "--dir",
		Short:   "Directory with migration files",
		Default: "./migrations",
	},
	"exclude": {
		Name:    "--exclude",
		Short:   "Exclude migrations by filename, comma separated",
		Default: "",
	},
	"help": {
		Name:    "--help",
		Short:   "Display help",
		Default: "",
	},
	"json": {
		Name:    "--json",
		Short:   "Format output as JSON",
		Default: "",
	},
	"lock-mode": {
		Name:     "--lock-mode",
		Short:    "Set a lock mode",
		Availble: []string{"none", "advisory-session"},
		Default:  "none",
	},
	"no-versioning": {
		Name:    "--no-versioning",
		Short:   "Do not store version info in the database, just run the migrations",
		Default: "",
	},
	"table": {
		Name:    "--table",
		Short:   "Table name to store version info",
		Default: "goose_db_version",
	},
	"v": {
		Name:    "--v",
		Short:   "Turn on verbose mode",
		Default: "",
	},
}

const (
	upCmdUsage = `
The up command runs all available migrations.

USAGE
  goose up [flags]

FLAGS
  --allow-missing         Allow missing (out-of-order) migrations
  --dbstring              Database connection string
  --dir                   Directory with migration files (default: "./migrations")
  --exclude               Exclude migrations by filename, comma separated
  --help                  Display help
  --json                  Format output as JSON
  --lock-mode             Set a lock mode [none, advisory-session] (default: "none")
  --no-versioning         Do not store version info in the database, just run the migrations
  --table                 Table name to store version info (default: "goose_db_version")
  --v                     Turn on verbose mode

EXAMPLES
  $ goose up --dbstring="postgres://user:password@localhost:5432/dbname" --dir=db/migrations
  $ GOOSE_DIR=./examples/sql-migrations GOOSE_DBSTRING="sqlite:./test.db" goose up-to 3
`
)
