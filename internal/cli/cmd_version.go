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
		Name:      "version",
		FlagSet:   fs,
		UsageFunc: func(c *ffcli.Command) string { return versionCmdUsage },
		Exec:      execVersionCmd(root),
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

const (
	versionCmdUsage = `
Print the current version of the database.

Note, if using --allow-missing, this command will return the recently applied migration, rather than 
the highest applied migration by version.

USAGE
  goose version [flags]

FLAGS
  --allow-missing         Allow missing (out-of-order) migrations
  --dbstring              Database connection string
  --dir                   Directory with migration files (default: "./migrations")
  --exclude               Exclude migrations by filename, comma separated
  --help                  Display help
  --json                  Format output as JSON
  --lock-mode             Set a lock mode [none, advisory-session] (default: "none")
  --table                 Table name to store version info (default: "goose_db_version")
  --v                     Turn on verbose mode

EXAMPLES
  $ goose version --dbstring="postgres://user:password@localhost:5432/dbname" --dir=db/migrations
  $ GOOSE_DIR=./examples/sql-migrations GOOSE_DBSTRING="sqlite:./test.db" goose version
`
)
