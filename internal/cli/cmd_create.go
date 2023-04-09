package cli

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pressly/goose/v4"
)

type createCmd struct {
	root *rootConfig

	sequential bool
	noTx       bool
}

func newCreateCmd(root *rootConfig) *ffcli.Command {
	c := createCmd{root: root}
	fs := flag.NewFlagSet("goose create", flag.ExitOnError)
	fs.BoolVar(&c.sequential, "s", false, "use sequential versions")
	fs.BoolVar(&c.noTx, "no-tx", false, "do not wrap migration in a transaction")
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:      "create",
		FlagSet:   fs,
		UsageFunc: func(c *ffcli.Command) string { return createCmdUsage },
		Exec:      c.Exec,
	}
}

func (c *createCmd) Exec(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("goose create requires 2 arguments: {sql | go} <name>\n\nExample: goose create sql add users table")
	}

	// TODO(mf): It'd be nice if ffcli supported mixing flags and positional arguments on the
	// same level. See https://github.com/peterbourgon/ff/issues/100
	//
	// For now, we should handle this ourselves if it looks like a flag we know about. The
	// same applies to the other commands.

	var migrationType goose.MigrationType
	switch strings.ToLower(args[0]) {
	case "go":
		migrationType = goose.MigrationTypeGo
	case "sql":
		migrationType = goose.MigrationTypeSQL
	default:
		return fmt.Errorf(`invalid migration type: first argument must be one of "sql" or "go"\n\nExample: goose create sql add users table`)
	}

	name := strings.Join(args[1:], " ")
	options := &goose.CreateOptions{
		Sequential: c.sequential,
		NoTx:       c.noTx,
	}
	filename, err := goose.Create(c.root.dir, migrationType, name, options)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.root.stdout, "Created: %s\n", filename)
	return nil
}

const (
	createCmdUsage = `
Create a new .sql or .go migration file with boilerplate.

The name argument is used to generate the filename. The name is converted to snake_case and 
prepended with a timestamp (20230409090029_), or a sequential number (00009_) if -s is used.

If the directory does not exist, it will be created.

USAGE
  goose create [flags] {sql | go} <name>

FLAGS
  -dir        Path to migrations directory (default: "./migrations")
  -s          Use sequential versions
  -no-tx      Mark the file as not requiring a transaction

EXAMPLES
  $ goose -dir ./schema/migrations create sql add users table
  $ GOOSE_DIR=./data/schema/migrations goose create -s -no-tx go backfill_emails
`
)
