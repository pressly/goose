package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pressly/goose/v4"
)

type createCmd struct {
	root *rootConfig

	dir        string
	sequential bool
	noTx       bool
}

func newCreateCmd(root *rootConfig) *ffcli.Command {
	c := &createCmd{root: root}
	fs := flag.NewFlagSet("goose create", flag.ExitOnError)
	fs.BoolVar(&c.sequential, "s", false, "")
	fs.BoolVar(&c.noTx, "no-tx", false, "")
	fs.StringVar(&c.dir, "dir", "", "")

	usageOpt := &usageOpt{
		examples: []string{
			"$ goose create --dir=./schema/migrations sql add users table",
			"$ GOOSE_DIR=./data/schema/migrations goose create --s --no-tx go backfill_emails",
		},
	}
	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "goose create [flags] {sql | go} <name>",
		ShortHelp:  "Create a new .sql or .go migration file with boilerplate",
		LongHelp:   createCmdLongHelp,
		FlagSet:    fs,
		UsageFunc:  newUsageFunc(usageOpt),
		Exec:       c.Exec,
	}
}

func (c *createCmd) Exec(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("goose create requires 2 arguments: {sql | go} <name>\n\nExample: goose create sql add users table")
	}

	// TODO(mf): It'd be nice if ffcli supported mixing flags and positional arguments on the
	// same level. See https://github.com/peterbourgon/ff/issues/100

	var sourceType goose.SourceType
	switch strings.ToLower(args[0]) {
	case "go":
		sourceType = goose.SourceTypeGo
	case "sql":
		sourceType = goose.SourceTypeSQL
	default:
		return fmt.Errorf("invalid migration type: first argument must be one of sql or go\n\nExample: goose create sql add users table")
	}

	dir := coalesce(c.dir, GOOSE_DIR)
	if dir == "" {
		return fmt.Errorf("goose create requires a migrations directory: %w", errNoDir)
	}

	name := strings.Join(args[1:], " ")
	options := &goose.CreateOptions{
		Sequential: c.sequential,
		NoTx:       c.noTx,
	}
	filename, err := goose.Create(dir, sourceType, name, options)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Created: %s\n", filename)
	return nil
}

const createCmdLongHelp = `
Create a new .sql or .go migration file with boilerplate.

The name argument is used to generate the filename. The name is converted to snake_case and 
prepended with a timestamp (20230409090029_), or a sequential number (00009_) if -s is used.

If the directory does not exist, it will be created.
`
