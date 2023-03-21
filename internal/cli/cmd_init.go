package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/template"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pressly/goose/v4"
)

func newInitCmd(root *rootConfig) *ffcli.Command {
	fs := flag.NewFlagSet("goose init", flag.ExitOnError)
	var (
		dir        string
		sequential bool
	)
	fs.StringVar(&dir, "dir", "", "")
	fs.BoolVar(&sequential, "s", true, "")

	return &ffcli.Command{
		Name:       "init",
		ShortUsage: "goose init",
		ShortHelp:  "Initialize a new project with a sample migration",
		LongHelp:   initCmdLongHelp,
		FlagSet:    fs,
		UsageFunc:  newUsageFunc(nil),
		Exec: func(ctx context.Context, args []string) error {
			dir := coalesce(dir, GOOSE_DIR)
			if dir == "" {
				return fmt.Errorf("goose init requires a migrations directory: %w", errNoDir)
			}
			opt := &goose.CreateOptions{
				Sequential: sequential,
				Template:   gooseInitTmpl,
			}
			filename, err := goose.Create(dir, goose.SourceTypeSQL, "initial", opt)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Created: %s\n", filename)
			return nil
		},
	}
}

var gooseInitTmpl = template.Must(template.New("goose.sql-migration").Parse(`-- Thank you for giving goose a try!
-- 
-- This file was automatically created by running goose init. If you're familiar with goose
-- feel free to remove/rename this file, write some SQL and goose up.
-- 
-- Documentation can be found here: https://pressly.github.io/goose 
-- Blog post that covers writing .sql files: https://pressly.github.io/goose/blog/2022/overview-sql-file/
-- 
-- Briefly, a single .sql file holds both Up and Down migrations.
-- 
-- All .sql files are expected to have a -- +goose Up annotation.
-- The -- +goose Down annotation is optional, but recommended, and must come after the Up annotation.
-- 
-- The -- +goose NO TRANSACTION annotation may be added to the top of the file to run statements 
-- outside a transaction. Up and Down migrations within this file will run without a transaction.
-- 
-- More complex statements with semicolons must be annotated with the -- +goose StatementBegin 
-- and -- +goose StatementEnd annotations to be properly recognized.
-- 
-- Use GitHub issues for reporting bugs and requesting features, enjoy!

-- +goose Up
SELECT 'up SQL query';

-- +goose Down
SELECT 'down SQL query';
`))

const initCmdLongHelp = `
Initialize a new project with a sample migration.

If the direcotry does not exist, it will be created.
`
